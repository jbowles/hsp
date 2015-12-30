package main

// EanApi provides basic GET/POST parameter queries and resposne handling for
// EAN Hotel List APIs. It also supports both XML and JSON. However, EAN API does not support embedded json requests so the JSON usage is limited: XML should be the defualt for EAN as it seems the support XML encoded requests is much better.
// NOTE: Structs have the minimal required params per Hotel queries; there are many more available filters and request params.
// TODO: read configuration from a file.

import (
	"fmt"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/url"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/jbowles/hotel_supply_platform/hspservice"
	"github.com/jbowles/quicksilver/formatter"
)

// interface to satisfy interface methods
type EanHspService struct {
	Service                  hspservice.Hsp
	cid                      string
	minorRev                 string
	apiKey                   string
	locale                   string
	currencyCode             string // only for booking and payment type
	customerSessionId        string
	customerIpAddress        string
	customerUserAgent        string
	supplierCacheTolerance   string
	includeHotelFeeBreakdown string
	supplierType             string
	maxRatePlanCounter       string
	includeDetails           string
	options                  string
}

type eanLoggingMiddleware struct {
	hspservice.Hsp
	//hspservice.Hsp
	logger log.Logger
}

/*
func eanLoggingMiddleware(logger log.Logger) ServiceMiddleware {
	return func(next hspservice.Hsp) hspservice.Hsp {
		return logmw{logger, next}
	}
}
*/

type eanInstrumentingMiddleware struct {
	hspservice.Hsp
	//requestCount        metrics.Counter
	requestDuration metrics.TimeHistogram
	//rateBreakdownResult metrics.Histogram
}

// satisfy interface
func (EanHspService) RateBreakdown(rbreq hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	h := HotelAvail{Format: "xml"}
	//
	h.HotelId.List = []int{225697, 116908}
	h.RoomGroup.Rm = []Room{{NumberOfAdults: 2, NumberOfChildren: 0, ChildAges: []int{}}}

	rbreq.RequestUrl = hspservice.Build(&h, 14)
	rbres.Request = rbreq
	return
}

func (m eanLoggingMiddleware) RateBreakdown(rbreq hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	defer func(begin time.Time) {
		_ = m.logger.Log(
			"method", "ean_rate_breakdown",
			"took", time.Since(begin),
		)
	}(time.Now())

	//rbres = hspservice.RateBreakdownResponse{rbreq, nil}
	return
}

func (m eanInstrumentingMiddleware) RateBreakdown(rbreq hspservice.RateBreakdownRequest) (rbres hspservice.RateBreakdownResponse) {
	defer func(begin time.Time) {
		methodField := metrics.Field{Key: "method", Value: "ean_rate_breakdown"}
		errorField := metrics.Field{Key: "error", Value: fmt.Sprintf("%v", rbres.Error)}
		//m.requestCount.With(methodField).With(errorField).Add(1)
		m.requestDuration.With(methodField).With(errorField).Observe(time.Since(begin))
	}(time.Now())

	//rbres = hspservice.RateBreakdownResponse{rbreq, nil}
	return
}

const (
	hotelListPath = "http://api.ean.com/ean-services/rs/hotel/v3/list?"
	//roomAvailPath = "http://api.ean.com/ean-services/rs/hotel/v3/avail?"
)

// MakeEanSpecs is a convenience function for building static EanSpecs.
// It is format agnostic and called inside the Ean interface build() method.
func MakeEanSpecs() EanHspService {
	return EanHspService{
		cid:                      "",
		minorRev:                 "26",
		apiKey:                   "",
		locale:                   "en_US",
		customerSessionId:        "theother",
		customerIpAddress:        "that",
		customerUserAgent:        "this",
		currencyCode:             "USD",
		supplierCacheTolerance:   "MIN",
		includeHotelFeeBreakdown: "false",
		supplierType:             "E",
		maxRatePlanCounter:       "10",
		includeDetails:           "false",
		options:                  "ROOM_RATE_DETAILS",
	}
}

// HotelAvail is the toplevel struct for buidling EAN requests.
type HotelAvail struct {
	XMLName         xml.Name `xml:"HotelListRequest" json:"-"`
	HotelId                  //see HotelId
	Address                  //see Address
	ArrivalDate     string   `xml:"arrivalDate" json:"arrivalDate"`
	DepartDate      string   `xml:"departureDate" json:"departureDate"`
	RoomGroup       `xml:"RoomGroup" json:"-"`
	NumberOfResults int    `xml:"numberOfResults,omitempty" json:"numberOfResults,omitempty"` // range == [1,200], default == 20 //HOTEL
	Format          string `xml:"-" json:"-"`
	hspservice.Supplier
}

// HotelIdList contains list of hotel ids in EAN requests.
type HotelId struct {
	List []int `xml:"hotelIdList" json:"-"`
}

//Address sets fields for XML or JSON
type Address struct {
	City              string `xml:"city,omitempty" json:"city,omitempty"`
	StateProvinceCode string `xml:"stateProvinceCode,omitempty" json:"stateProvinceCode,omitempty"`
	CountryCode       string `xml:"countryCode,omitempty" json:"countryCode,omitempty"`
}

// RoomGroup is an EAN defined top level Class/container for room information.
// Support here is for both XML and JSON, as well allowing for more than one room request (i.e., slices).
// In multiple room requests we expect embedded fields.
type RoomGroup struct {
	Rm []Room
}

type Room struct {
	XMLName          xml.Name `xml:"Room" json:"-"`
	NumberOfAdults   int      `xml:"numberOfAdults" json:"numberOfAdults,int"`
	NumberOfChildren int      `xml:"numberOfChildren,omitempty" json:"numberOfChildren,omitempty"`
	ChildAges        []int    `xml:"childrenAges,omitempty" json:"childrenAges,omitempty"`
}

// DateRange implements Ean interface. It builds a date range for arrival and departure.
func (h *HotelAvail) DateRange(days int) {
	a := time.Now()
	d := a.AddDate(0, 0, days)
	//h.ArrivalDate = a.Format(dateLayout)
	//h.DepartDate = d.Format(dateLayout)

	h.ArrivalDate, h.DepartDate = formatter.TimeInStringsOut(formatter.EanDateLayout, a, d)
}

// encode provides xml/json Marshalling and returns the formatted bytes.
// XML is the default, as the Ean API has better support for XML.
func (h *HotelAvail) encode() *bytes.Buffer {
	switch h.Format {
	case "xml":
		buff, _ := xml.Marshal(h)
		return bytes.NewBuffer(buff)
	case "json":
		buff, _ := json.Marshal(h)
		return bytes.NewBuffer(buff)
	default:
		buff, _ := xml.Marshal(h)
		return bytes.NewBuffer(buff)
	}
}

// Params implements Supplier interface. It creates the url with common key-values as well as
// query params that will be used for making the Ean request.
// XML is the default format for query params, as the Ean API has better support for XML.
// TODO: make the json encoding to URL format cleaner, especially want it to automatically drop empty fields and pick up popuated fields and then format them as '&key=value&'... as of now DO NOT use the JSON configuration because EAN API does not support requests in JSON.
func (h *HotelAvail) Params() *url.URL {
	r, _ := url.Parse(hotelListPath)
	v := r.Query()
	e := MakeEanSpecs()

	enc := h.encode()
	//log.Printf("Format: %q, Supplier: %q Adults: %v Rooms: %v Arrival %q Depart %q Domain: %q\n", h.Format, h.Supplier, h.RoomGroup.Rm[0].NumberOfAdults, len(h.RoomGroup.Rm), h.ArrivalDate, h.DepartDate, r)
	enc_to_byte, _ := ioutil.ReadAll(enc)

	v.Add("cid", e.cid)
	v.Add("minorRev", e.minorRev)
	v.Add("apiKey", e.apiKey)
	v.Add("locale", e.locale)
	v.Add("currencyCode", e.currencyCode)
	v.Add("supplierCacheTolerance", e.supplierCacheTolerance)
	v.Add("includeHotelFeeBreakdown", e.includeHotelFeeBreakdown)
	v.Add("supplierType", e.supplierType)
	v.Add("maxRatePlanCounter", e.maxRatePlanCounter)
	v.Add("includeDetails", e.includeDetails)
	v.Add("options", e.options)
	switch h.Format {
	case "xml":
		v.Add("xml", string(enc_to_byte))
	case "json":
		v.Add("arrivalDate", h.ArrivalDate)
		v.Add("departureDate", h.DepartDate)
		hotelList := []string{}
		for i := 0; i < len(h.HotelId.List); i++ {
			hotelList = append(hotelList, strconv.Itoa(h.HotelId.List[i]))
		}
		v.Add("hotelIdList", strings.Join(hotelList, ","))
		for i := 0; i < len(h.RoomGroup.Rm); i++ {
			v.Add(("room" + strconv.Itoa(i+1)), strconv.Itoa(h.RoomGroup.Rm[i].NumberOfAdults))
		}
	default:
		v.Add("xml", string(enc_to_byte))
	}
	r.RawQuery = v.Encode()
	return r
}
