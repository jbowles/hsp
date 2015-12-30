package main

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/jbowles/hotel_supply_platform/hspservice"
)

type ServiceMiddleware func(hspservice.Hsp) hspservice.Hsp

type loggingMiddleware struct {
	hspservice.Hsp
	logger log.Logger
}
type instrumentingMiddleware struct {
	hspservice.Hsp
	//requestCount        metrics.Counter
	requestDuration metrics.TimeHistogram
	//rateBreakdownResult metrics.Histogram
}

// interface to satisfy interface methods
type HspService struct{}

// satisfy interface
func (HspService) RateBreakdown(hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	return rbres
}

func (m loggingMiddleware) RateBreakdown(rbreq hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	defer func(begin time.Time) {
		_ = m.logger.Log(
			"method", "rate_breakdown",
			"took", time.Since(begin),
		)
	}(time.Now())

	//rbres = hspservice.RateBreakdownResponse{rbreq, nil}
	return
}

func (m instrumentingMiddleware) RateBreakdown(rbreq hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	defer func(begin time.Time) {
		methodField := metrics.Field{Key: "method", Value: "rate_breakdown"}
		errorField := metrics.Field{Key: "error", Value: fmt.Sprintf("%v", rbres.Error)}
		//m.requestCount.With(methodField).With(errorField).Add(1)
		m.requestDuration.With(methodField).With(errorField).Observe(time.Since(begin))
	}(time.Now())

	rbres = hspservice.RateBreakdownResponse{rbreq, nil}
	return
}
