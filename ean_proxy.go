package main

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/jbowles/hotel_supply_platform/hspservice"
	jujuratelimit "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/loadbalancer"
	"github.com/go-kit/kit/loadbalancer/static"
	"github.com/go-kit/kit/log"
	kitratelimit "github.com/go-kit/kit/ratelimit"
	httptransport "github.com/go-kit/kit/transport/http"
)

// proxymw implements StringService, forwarding Uppercase requests to the
// provided endpoint, and serving all other (i.e. Count) requests via the
// embedded StringService.
type proxymw struct {
	context.Context
	EanHspService endpoint.Endpoint
	hspservice.Hsp
}

func eanProxyingMiddleware(proxyList string, ctx context.Context, logger log.Logger) ServiceMiddleware {
	if proxyList == "" {
		logger.Log("proxy_to", "none")
		return func(next hspservice.Hsp) hspservice.Hsp { return next }
	}
	proxies := split(proxyList)
	logger.Log("proxy_to", fmt.Sprint(proxies))

	return func(next hspservice.Hsp) hspservice.Hsp {
		var (
			qps         = 100 // max to each instance
			publisher   = static.NewPublisher(proxies, factory(ctx, qps), logger)
			lb          = loadbalancer.NewRoundRobin(publisher)
			maxAttempts = 3
			maxTime     = 100 * time.Millisecond
			endpoint    = loadbalancer.Retry(maxAttempts, maxTime, lb)
		)
		return proxymw{ctx, endpoint, next}
	}
}

// satisfy interface
func (proxymw) RateBreakdown(rbreq hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	h := HotelAvail{Format: "xml"}
	//
	h.HotelId.List = []int{225697, 116908}
	h.RoomGroup.Rm = []Room{{NumberOfAdults: 2, NumberOfChildren: 0, ChildAges: []int{}}}

	rbreq.RequestUrl = hspservice.Build(&h, 14)
	rbres.Request = rbreq
	return
}

func factory(ctx context.Context, qps int) loadbalancer.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		var e endpoint.Endpoint
		e = makeEanProxy(ctx, instance)
		e = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(e)
		e = kitratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(float64(qps), int64(qps)))(e)
		return e, nil, nil
	}
}

func makeEanProxy(ctx context.Context, instance string) endpoint.Endpoint {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		panic(err)
	}
	if u.Path == "" {
		u.Path = "/ean/rate_breakdown"
	}
	return httptransport.NewClient(
		"GET",
		u,
		hspservice.EncodeRateBreakdownRequest,
		hspservice.DecodeRateBreakdownResponse,
	).Endpoint()
}

func split(s string) []string {
	a := strings.Split(s, ",")
	for i := range a {
		a[i] = strings.TrimSpace(a[i])
	}
	return a
}
