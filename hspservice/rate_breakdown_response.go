package hspservice

// RateBreakdownResponse is the business domain type for a RateBreakdownService method response.
type RateBreakdownResponse struct {
	Request RateBreakdownRequest
	Error   error `json:"error"`
}
