// Copyright (c) TrueTickets, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

// SubscriptionPrice represents a per-territory price of a subscription in the
// App Store Connect API.
type SubscriptionPrice struct {
	Type          string                          `json:"type"`
	ID            string                          `json:"id"`
	Attributes    SubscriptionPriceAttributes     `json:"attributes"`
	Relationships *SubscriptionPriceRelationships `json:"relationships,omitempty"`
}

// SubscriptionPriceAttributes represents the attributes of a subscription
// price.
type SubscriptionPriceAttributes struct {
	StartDate            string `json:"startDate,omitempty"`
	PreserveCurrentPrice bool   `json:"preserveCurrentPrice,omitempty"`
}

// SubscriptionPriceRelationships represents the relationships of a subscription
// price.
type SubscriptionPriceRelationships struct {
	SubscriptionPricePoint *Relationship `json:"subscriptionPricePoint,omitempty"`
	Territory              *Relationship `json:"territory,omitempty"`
}

// SubscriptionPriceCreateRequest represents the request body for creating a
// subscription price.
type SubscriptionPriceCreateRequest struct {
	Data SubscriptionPriceCreateRequestData `json:"data"`
}

// SubscriptionPriceCreateRequestData represents the data for creating a
// subscription price.
type SubscriptionPriceCreateRequestData struct {
	Type          string                                      `json:"type"`
	Attributes    SubscriptionPriceCreateRequestAttributes    `json:"attributes"`
	Relationships SubscriptionPriceCreateRequestRelationships `json:"relationships"`
}

// SubscriptionPriceCreateRequestAttributes represents the attributes for
// creating a subscription price.
type SubscriptionPriceCreateRequestAttributes struct {
	StartDate            *string `json:"startDate,omitempty"`
	PreserveCurrentPrice *bool   `json:"preserveCurrentPrice,omitempty"`
}

// SubscriptionPriceCreateRequestRelationships represents the relationships for
// creating a subscription price.
type SubscriptionPriceCreateRequestRelationships struct {
	Subscription           RelationshipOne `json:"subscription"`
	SubscriptionPricePoint RelationshipOne `json:"subscriptionPricePoint"`
	Territory              RelationshipOne `json:"territory"`
}

// SubscriptionPriceResponse represents the response from the subscription price
// API.
type SubscriptionPriceResponse struct {
	Data  SubscriptionPrice `json:"data"`
	Links Links             `json:"links,omitempty"`
}
