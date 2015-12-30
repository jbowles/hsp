package hspservice

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// DecodeRateBreakdownRequest decodes the request from the provided HTTP request, simply
// by JSON decoding from the request body. It's designed to be used in
// transport/http.Server.
func DecodeRateBreakdownRequest(r *http.Request) (interface{}, error) {
	var request RateBreakdownRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	return request, err
}

// EncodeRateBreakdownRequest encodes the request to the provided HTTP request, simply
// by JSON encoding to the request body. It's designed to be used in
// transport/http.Client.
func EncodeRateBreakdownRequest(r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// DecodeRateBreakdownResponse decodes the response from the provided HTTP response,
// simply by JSON decoding from the response body. It's designed to be used in
// transport/http.Client.
func DecodeRateBreakdownResponse(resp *http.Response) (interface{}, error) {
	var response RateBreakdownResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

// EncodeRateBreakdownResponse encodes the response to the provided HTTP response
// writer, simply by JSON encoding to the writer. It's designed to be used in
// transport/http.Server.
func EncodeRateBreakdownResponse(w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
