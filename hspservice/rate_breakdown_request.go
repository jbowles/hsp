package hspservice

import "net/url"

// RateBreakdownRequest is the business domain type for a RateBreakdown method request.
type RateBreakdownRequest struct {
	//Arrival   time.Time `json:"arrival"`
	//Departure time.Time `json:"departure"`
	RequestUrl *url.URL
	Arrival    string `json:"arrival"`
	Departure  string `json:"departure"`
	Currency   string `json:"currency"`
}
