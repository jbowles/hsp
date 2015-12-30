package main

import (
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/jbowles/hotel_supply_platform/hspservice"
)

func makeRateBreakdownEndpoint(svc hspservice.Hsp) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(hspservice.RateBreakdownRequest)
		result := svc.RateBreakdown(
			hspservice.RateBreakdownRequest{
				req.RequestUrl,
				req.Arrival,
				req.Departure,
				req.Currency,
			},
		)
		return result, nil
	}
}

func makeEanRateBreakdownEndpoint(svc EanHspService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(hspservice.RateBreakdownRequest)
		result := svc.RateBreakdown(
			hspservice.RateBreakdownRequest{
				req.RequestUrl,
				req.Arrival,
				req.Departure,
				req.Currency,
			},
		)
		return result, nil
	}
}
