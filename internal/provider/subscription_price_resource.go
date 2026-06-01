// Copyright (c) TrueTickets, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SubscriptionPriceResource{}
var _ resource.ResourceWithImportState = &SubscriptionPriceResource{}

// NewSubscriptionPriceResource creates a new subscription price resource.
func NewSubscriptionPriceResource() resource.Resource {
	return &SubscriptionPriceResource{}
}

// SubscriptionPriceResource defines the resource implementation.
type SubscriptionPriceResource struct {
	client *Client
}

// SubscriptionPriceResourceModel describes the resource data model.
type SubscriptionPriceResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	SubscriptionID           types.String `tfsdk:"subscription_id"`
	SubscriptionPricePointID types.String `tfsdk:"subscription_price_point_id"`
	Territory                types.String `tfsdk:"territory"`
	StartDate                types.String `tfsdk:"start_date"`
	PreserveCurrentPrice     types.Bool   `tfsdk:"preserve_current_price"`
}

func (r *SubscriptionPriceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription_price"
}

func (r *SubscriptionPriceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the price of a subscription in a single territory. Subscriptions are priced per territory (there is no " +
			"single price schedule like In-App Purchases), so create one resource per territory. Price point IDs are resolved with the " +
			"`appleappstoreconnect_subscription_price_point` data source. Changing any attribute replaces the price.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the subscription price.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the subscription this price applies to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subscription_price_point_id": schema.StringAttribute{
				MarkdownDescription: "The price point ID for this territory (resolve it with the `appleappstoreconnect_subscription_price_point` data source). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"territory": schema.StringAttribute{
				MarkdownDescription: "The territory code this price applies to (e.g. `USA`). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"start_date": schema.StringAttribute{
				MarkdownDescription: "The date the price takes effect (ISO 8601, e.g. `2025-06-01`). Omit for immediate effect. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"preserve_current_price": schema.BoolAttribute{
				MarkdownDescription: "When `true`, existing subscribers keep their current price and only new/renewing subscribers get this price. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *SubscriptionPriceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *SubscriptionPriceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubscriptionPriceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := SubscriptionPriceCreateRequestAttributes{}
	if !data.StartDate.IsNull() {
		v := data.StartDate.ValueString()
		attrs.StartDate = &v
	}
	if !data.PreserveCurrentPrice.IsNull() && !data.PreserveCurrentPrice.IsUnknown() {
		v := data.PreserveCurrentPrice.ValueBool()
		attrs.PreserveCurrentPrice = &v
	}

	createReq := SubscriptionPriceCreateRequest{
		Data: SubscriptionPriceCreateRequestData{
			Type:       "subscriptionPrices",
			Attributes: attrs,
			Relationships: SubscriptionPriceCreateRequestRelationships{
				Subscription: RelationshipOne{
					Data: RelationshipData{Type: "subscriptions", ID: data.SubscriptionID.ValueString()},
				},
				SubscriptionPricePoint: RelationshipOne{
					Data: RelationshipData{Type: "subscriptionPricePoints", ID: data.SubscriptionPricePointID.ValueString()},
				},
				Territory: RelationshipOne{
					Data: RelationshipData{Type: "territories", ID: data.Territory.ValueString()},
				},
			},
		},
	}

	tflog.Debug(ctx, "Creating subscription price", map[string]any{
		"subscription_id": data.SubscriptionID.ValueString(),
		"territory":       data.Territory.ValueString(),
	})

	apiResp, err := r.client.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/v1/subscriptionPrices",
		Body:     createReq,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create subscription price, got error: %s", err),
		)
		return
	}

	var price SubscriptionPrice
	if err := json.Unmarshal(apiResp.Data, &price); err != nil {
		resp.Diagnostics.AddError(
			"Parse Error",
			fmt.Sprintf("Unable to parse subscription price response, got error: %s", err),
		)
		return
	}

	if price.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid API Response",
			"The API response did not contain a valid ID for the created subscription price",
		)
		return
	}

	data.ID = types.StringValue(price.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubscriptionPriceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubscriptionPriceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no GET for an individual subscription price; list the prices of
	// the subscription and find this one by ID.
	elements, err := doPaginated(ctx, r.client, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/v1/subscriptions/%s/prices", data.SubscriptionID.ValueString()),
		Query:    map[string]string{"limit": "200"},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read subscription prices, got error: %s", err),
		)
		return
	}

	found := false
	for _, element := range elements {
		var price SubscriptionPrice
		if err := json.Unmarshal(element, &price); err != nil {
			resp.Diagnostics.AddError(
				"Parse Error",
				fmt.Sprintf("Unable to parse subscription price, got error: %s", err),
			)
			return
		}

		if price.ID != data.ID.ValueString() {
			continue
		}

		found = true
		if price.Relationships != nil {
			if price.Relationships.SubscriptionPricePoint != nil && price.Relationships.SubscriptionPricePoint.Data != nil {
				data.SubscriptionPricePointID = types.StringValue(price.Relationships.SubscriptionPricePoint.Data.ID)
			}
			if price.Relationships.Territory != nil && price.Relationships.Territory.Data != nil {
				data.Territory = types.StringValue(price.Relationships.Territory.Data.ID)
			}
		}
		// start_date and preserve_current_price are creation-time directives
		// (RequiresReplace) and are kept from state to avoid format round-trip
		// drift.
		break
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubscriptionPriceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes are RequiresReplace, so Update is never reached with a
	// real change. Implemented to satisfy the interface.
	var data SubscriptionPriceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubscriptionPriceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubscriptionPriceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/v1/subscriptionPrices/%s", data.ID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete subscription price, got error: %s", err),
		)
		return
	}
}

func (r *SubscriptionPriceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The price has no individual GET endpoint, so import needs the parent
	// subscription ID: "<subscription_id>:<price_id>".
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the form 'subscription_id:price_id', got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subscription_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
