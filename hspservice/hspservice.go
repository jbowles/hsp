package hspservice

import "net/url"

// Hsp is the abstract representation of the HotelSupplyPlatform service
type Hsp interface {
	RateBreakdown(r RateBreakdownRequest) RateBreakdownResponse
	//ProviderSelection([]string) ([]string, error)
	//Auction()
	//RateValidation()
	//HotelRateSearch()
}

// Affiliate interface defines two methods for all affiliate APIs.
// Params builds and returns the full URI needed for making query.
// DateRange defines the range of dates for request.
type Supplier interface {
	Params() *url.URL
	DateRange(days int)
}

// Build is an exported function that implements the Affiliate interface.
// It accepts and number of days from current date to define date range and returns
// the URL needed to make the request.
func Build(supplier Supplier, days int) *url.URL {
	supplier.DateRange(days)
	return supplier.Params()
}
