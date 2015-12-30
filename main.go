package main

import (
	"flag"
	"fmt"
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jbowles/hotel_supply_platform/hspservice"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/expvar"
	"github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return fmt.Errorf("%s", <-c)
}

func main() {
	// Flag domain. Note that gRPC transitively registers flags via its import
	// of glog. So, we define a new flag set, to keep those domains distinct.
	fs := flag.NewFlagSet("", flag.ExitOnError)
	var (
		eanHttpAddr = fs.String("ean.addr", ":8001", "Address for Ean HTTP (JSON) server")
		httpAddr    = fs.String("http.addr", ":8022", "Address for HTTP (JSON) server")
		debugAddr   = fs.String("debug.addr", ":8000", "Address for HTTP debug/instrumentation server")
	)
	flag.Usage = fs.Usage // only show our flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	// package log
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC).With("caller", log.DefaultCaller)
		stdlog.SetFlags(0)                             // flags are handled by Go kit's logger
		stdlog.SetOutput(log.NewStdlibAdapter(logger)) // redirect anything using stdlib log to us
	}

	// package metrics
	var requestDuration metrics.TimeHistogram
	{
		requestDuration = metrics.NewTimeHistogram(time.Nanosecond, metrics.NewMultiHistogram(
			expvar.NewHistogram("request_duration_ns", 0, 5e9, 1, 50, 95, 99),
			prometheus.NewSummary(stdprometheus.SummaryOpts{
				Namespace: "partnerFusion",
				Subsystem: "hps_service",
				Name:      "duration_ns",
				Help:      "Request duration in nanoseconds.",
			}, []string{"method"}),
		))
	}

	// Mechanical stuff
	rand.Seed(time.Now().UnixNano())
	root := context.Background()
	errc := make(chan error)

	// Business domain
	var svc hspservice.Hsp
	{
		svc = HspService{}
		svc = instrumentingMiddleware{svc, requestDuration}
		//svc = loggingMiddleware{svc, logger}
		svc = eanLoggingMiddleware{svc, logger}
	}

	go func() {
		errc <- interrupt()
	}()

	// Debug/instrumentation
	go func() {
		transportLogger := log.NewContext(logger).With("transport", "debug")
		transportLogger.Log("addr", *debugAddr)
		errc <- http.ListenAndServe(*debugAddr, nil) // DefaultServeMux
	}()

	// Transport: Ean HTTP/JSON client servers come first
	go func() {
		var (
			transportLogger = log.NewContext(logger).With("transport", "EAN-HTTP/JSON")
			mux             = http.NewServeMux()
			eanrateb        endpoint.Endpoint
		)

		eanrateb = makeEanRateBreakdownEndpoint(EanHspService{})
		mux.Handle("/ean/rate_breakdown", httptransport.NewServer(
			root,
			eanrateb,
			hspservice.DecodeRateBreakdownRequest,
			hspservice.EncodeRateBreakdownResponse,
			//httptransport.ServerBefore(traceSum),
			httptransport.ServerErrorLogger(transportLogger),
		))

		transportLogger.Log("addr", *eanHttpAddr)
		errc <- http.ListenAndServe(*eanHttpAddr, mux)
	}()

	// Transport: HTTP/JSON
	go func() {
		var (
			transportLogger = log.NewContext(logger).With("transport", "HTTP/JSON")
			mux             = http.NewServeMux()
			rateb           endpoint.Endpoint
		)

		rateb = makeRateBreakdownEndpoint(svc)
		mux.Handle("/rate_breakdown", httptransport.NewServer(
			root,
			rateb,
			//makeRateBreakdownEndpoint(svc),
			hspservice.DecodeRateBreakdownRequest,
			hspservice.EncodeRateBreakdownResponse,
			//httptransport.ServerBefore(traceSum),
			httptransport.ServerErrorLogger(transportLogger),
		))

		transportLogger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, mux)
	}()

	//Proxy to running servers
	/*
		go func() {
			svc = eanProxyingMiddleware(*eanHttpAddr, root, logger)(svc)
		}()
	*/

	logger.Log("fatal", <-errc)
}

/// one way to setup the server... simplified with one service and one api endpoint
/*
func main() {
	ctx := context.Background()
	logger := log.NewLogfmtLogger(os.Stderr)

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounter(stdprometheus.CounterOpts{
		Namespace: "pf_group",
		Subsystem: "hsp_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestDuration := metrics.NewTimeHistogram(time.Microsecond, kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "pf_group",
		Subsystem: "hsp_service",
		Name:      "request_duration_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys))
	rateBreakdownResult := kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "pf_group",
		Subsystem: "hsp_service",
		Name:      "rate_breakdown_result",
		Help:      "The result of each count method.",
	}, []string{}) // no fields here

	var svc hspservice.Hsp
	{
		svc = HspService{}
		svc = EanHspService{}
		svc = instrumentingMiddleware{svc, requestCount, requestDuration, rateBreakdownResult}
		svc = loggingMiddleware{svc, logger}
	}

	rateBreakdownHandler := httptransport.NewServer(
		ctx,
		makeRateBreakdownEndpoint(svc),
		hspservice.DecodeRateBreakdownRequest,
		hspservice.EncodeRateBreakdownResponse,
	)

	http.Handle("/rate_breakdown", rateBreakdownHandler)
	http.Handle("/metrics", stdprometheus.Handler())
	logger.Log("msg", "HTTP", "addr", ":8080")
	logger.Log("err", http.ListenAndServe(":8080", nil))
}
*/
