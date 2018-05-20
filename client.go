package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
)

type Response struct {
	*http.Response
}

func newResponse(r *http.Response) *Response {
	return &Response{Response: r}
}

// ErrorResponse represents an API error response
// with error fields populated as a string maps
type ErrorResponse struct {
	Response *http.Response
	Errors   map[string]json.RawMessage
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.Errors)
}

// Client represents an API client instance
// with service actions accessed as members
type Client struct {
	client  *http.Client
	BaseURL *url.URL
	common  service
	Echo    *EchoService
}

func withContext(ctx context.Context, req *http.Request) *http.Request {
	return req.WithContext(ctx)
}

// NewRequest creates an API request
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// Do performs an HTTP request
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = withContext(ctx, req)
	resp, err := c.client.Do(req)
	if err != nil {
		// Get context error upon cancelling
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		// Sanitize URL errors
		if e, ok := err.(*url.Error); ok {
			if url, err := url.Parse(e.URL); err == nil {
				e.URL = url.String()
				return nil, e
			}
		}
		return nil, err
	}
	response := newResponse(resp)
	defer resp.Body.Close()
	if err := CheckResponse(resp); err != nil {
		log.Println(err)
		return response, err
	}
	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			decErr := json.NewDecoder(resp.Body).Decode(v)
			if decErr == io.EOF {
				// Ignore EOF errors due to empty responses
				decErr = nil
			}
			if decErr != nil {
				err = decErr
			}
		}
	}
	return response, err
}

func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse.Errors)
	}
	return errorResponse
}

func resourceByName(c *Client, resource string) (interface{}, error) {
	clientR := reflect.ValueOf(c)
	clientF := reflect.Indirect(clientR).FieldByName(resource)
	if !clientF.IsValid() {
		return nil, errors.New("Resource not found")
	}
	return clientF.Interface(), nil
}

func methodByName(i interface{}, methodName string) (reflect.Value, error) {
	var ptr reflect.Value
	var value reflect.Value
	var finalMethod reflect.Value
	value = reflect.ValueOf(i)
	if value.Type().Kind() == reflect.Ptr {
		ptr = value
		value = ptr.Elem()
	} else {
		ptr = reflect.New(reflect.TypeOf(i))
		temp := ptr.Elem()
		temp.Set(value)
	}
	method := value.MethodByName(methodName)
	if method.IsValid() {
		finalMethod = method
	}
	method = ptr.MethodByName(methodName)
	if method.IsValid() {
		finalMethod = method
	}
	if finalMethod.IsValid() {
		return finalMethod, nil
	}
	return reflect.ValueOf(nil), errors.New("Method not found")
}

func (c *Client) CallMethodByName(
	resource string,
	method string,
	payload json.RawMessage,
) (
	*json.RawMessage,
	*Response,
	error,
) {
	resourceObj, err := resourceByName(c, resource)
	if err != nil {
		log.Fatal(err)
	}
	methodFunc, err := methodByName(resourceObj, method)
	if err != nil {
		log.Fatal(err)
	}
	in := []reflect.Value{
		reflect.ValueOf(context.Background()),
		reflect.ValueOf(payload),
	}
	results := methodFunc.Call(in)
	data := results[0].Interface().(*json.RawMessage)
	response := results[1].Interface().(*Response)
	return data, response, nil
}

func NewClient() *Client {
	httpClient := http.DefaultClient
	baseURL, _ := url.Parse(defaultBaseURL)
	c := &Client{
		client:  httpClient,
		BaseURL: baseURL,
	}
	c.common.client = c
	c.Echo = (*EchoService)(&c.common)
	return c
}
