package requests

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// Map is an alias for a map with string keys and any type values.
type Map map[string]any

// Files represents a slice of FileField, used for multipart file uploads.
type Files []FileField

// FileField describes a file field for multipart form data.
type FileField struct {
	FieldName string   // the field name in the form
	FileName  string   // the name of the file
	File      *os.File // a pointer to the file (if already opened)
	FilePath  string   // file path to open if File is nil
}

// RetryConfig holds configuration for retrying a request.
type RetryConfig struct {
	Count int           // number of retry attempts
	Delay time.Duration // delay between retry attempts
}

// retryConfigKey is an unexported type used as the key for storing retry configuration in the request context.
type retryConfigKey struct{}

// closeFile safely closes the provided file.
func closeFile(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

// ensureTransport makes sure the HTTP client has a Transport and returns it.
func ensureTransport(client *http.Client) *http.Transport {
	if client.Transport == nil {
		client.Transport = &http.Transport{}
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		tr = &http.Transport{}
		client.Transport = tr
	}
	return tr
}

// RequestOption defines a function type that can modify an HTTP request and/or client.
type RequestOption func(*http.Request, *http.Client) error

// WithStringBody sets a plain text body for the request.
func WithStringBody(body string) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		req.Body = io.NopCloser(strings.NewReader(body))
		req.ContentLength = int64(len([]byte(body)))
		req.Header.Set("Content-Type", "text/plain")
		return nil
	}
}

// WithXMLBody marshals the given data to XML and sets it as the request body.
func WithXMLBody(body any) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		xmlData, err := xml.Marshal(body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewReader(xmlData))
		req.ContentLength = int64(len(xmlData))
		req.Header.Set("Content-Type", "application/xml")
		return nil
	}
}

// WithJSONBody marshals the given data to JSON and sets it as the request body.
func WithJSONBody(body any) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewReader(jsonData))
		req.ContentLength = int64(len(jsonData))
		req.Header.Set("Content-Type", "application/json")
		return nil
	}
}

// WithFormBody encodes the provided map as URL-encoded form data and sets it as the request body.
func WithFormBody(data Map) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		formData := url.Values{}
		for key, value := range data {
			formData.Set(key, fmt.Sprintf("%v", value))
		}
		encodedData := formData.Encode()
		req.Body = io.NopCloser(strings.NewReader(encodedData))
		req.ContentLength = int64(len(encodedData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return nil
	}
}

// WithMultipartFiles constructs a multipart/form-data body with provided files and additional form fields.
func WithMultipartFiles(files Files, otherFields Map) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		var err error
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// Add file fields to multipart body.
		for _, fileField := range files {
			file := fileField.File
			if file == nil && fileField.FilePath == "" {
				continue
			}
			if file == nil && fileField.FilePath != "" {
				file, err = os.Open(fileField.FilePath)
				if err != nil {
					return err
				}
			}
			part, err := writer.CreateFormFile(fileField.FieldName, fileField.FileName)
			if err != nil {
				closeFile(file)
				return err
			}
			if _, err := io.Copy(part, file); err != nil {
				closeFile(file)
				return err
			}
			closeFile(file)
		}
		// Add other non-file fields.
		for key, value := range otherFields {
			if err = writer.WriteField(key, fmt.Sprintf("%v", value)); err != nil {
				return err
			}
		}
		// Close the writer to finalize the multipart body.
		if err := writer.Close(); err != nil {
			return err
		}
		req.Body = io.NopCloser(body)
		req.ContentLength = int64(body.Len())
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return nil
	}
}

// WithHeaders sets the specified headers on the HTTP request.
func WithHeaders(headers Map) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		for key, value := range headers {
			// Special handling for Host and Transfer-Encoding.
			if strings.ToLower(strings.TrimSpace(key)) == "host" {
				req.Host = fmt.Sprintf("%v", value)
			} else if strings.ToLower(strings.TrimSpace(key)) == "transfer-encoding" {
				req.TransferEncoding = []string{fmt.Sprintf("%v", value)}
			} else {
				req.Header.Set(key, fmt.Sprintf("%v", value))
			}
		}
		return nil
	}
}

// WithTimeout sets the timeout duration on the HTTP client.
func WithTimeout(timeout time.Duration) RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		client.Timeout = timeout
		return nil
	}
}

// WithDisableRedirect disables automatic redirection in the HTTP client.
func WithDisableRedirect() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

// WithDisableGzip disables automatic gzip decompression on the HTTP client.
func WithDisableGzip() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		tr := ensureTransport(client)
		tr.DisableCompression = true
		return nil
	}
}

// WithProxy configures the HTTP client to use the specified proxy URL.
func WithProxy(proxyURL string) RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return err
		}
		tr := ensureTransport(client)
		tr.Proxy = http.ProxyURL(proxy)
		return nil
	}
}

// WithSkipSSLVerify instructs the HTTP client to skip SSL/TLS certificate verification.
func WithSkipSSLVerify() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		tr := ensureTransport(client)
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
		return nil
	}
}

// WithHTTP2 forces the HTTP client to attempt HTTP/2 connections.
func WithHTTP2() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		tr := ensureTransport(client)
		tr.ForceAttemptHTTP2 = true
		return nil
	}
}

// WithCookieJar sets a new cookie jar for the HTTP client.
func WithCookieJar() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return err
		}
		client.Jar = jar
		return nil
	}
}

// WithBasicAuth sets HTTP basic authentication headers using the provided credentials.
func WithBasicAuth(username, password string) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		req.SetBasicAuth(username, password)
		return nil
	}
}

// WithRetryConfig attaches a RetryConfig to the request context for retry logic.
func WithRetryConfig(cfg RetryConfig) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		ctx := context.WithValue(req.Context(), retryConfigKey{}, cfg)
		*req = *req.WithContext(ctx)
		return nil
	}
}

// getRetryConfig extracts the RetryConfig from the provided options.
// If no retry configuration is set, it returns a configuration with Count 0 and Delay 0.
func getRetryConfig(method, url string, options []RequestOption) RetryConfig {
	req, _ := http.NewRequest(method, url, nil)
	dummyClient := http.DefaultClient
	var cfg RetryConfig
	var set bool
	for _, opt := range options {
		_ = opt(req, dummyClient)
		if rc, ok := req.Context().Value(retryConfigKey{}).(RetryConfig); ok {
			cfg = rc
			set = true
			break
		}
	}
	if !set {
		cfg = RetryConfig{Count: 0, Delay: 0}
	}
	return cfg
}

// RequestWithRetry executes an HTTP request with retry logic based on the attached RetryConfig.
// If no retry configuration is set (Count == 0), it returns an error and does not execute any request.
func RequestWithRetry(method, url string, options ...RequestOption) (*Response, []error) {
	cfg := getRetryConfig(method, url, options)
	if cfg.Count == 0 {
		return nil, []error{errors.New("no retry configuration set, no request executed")}
	}
	var errs []error
	var resp *Response
	for i := 0; i < cfg.Count; i++ {
		response, err := Request(method, url, options...)
		if err == nil {
			resp = response
			break
		}
		errs = append(errs, err)
		time.Sleep(cfg.Delay)
	}
	return resp, errs
}

// Convenience methods with retry logic.

// HeadWithRetry sends an HTTP HEAD request with retry logic.
func HeadWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodHead, url, options...)
}

// OptionsWithRetry sends an HTTP OPTIONS request with retry logic.
func OptionsWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodOptions, url, options...)
}

// GetWithRetry sends an HTTP GET request with retry logic.
func GetWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodGet, url, options...)
}

// PostWithRetry sends an HTTP POST request with retry logic.
func PostWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodPost, url, options...)
}

// PutWithRetry sends an HTTP PUT request with retry logic.
func PutWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodPut, url, options...)
}

// PatchWithRetry sends an HTTP PATCH request with retry logic.
func PatchWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodPatch, url, options...)
}

// DeleteWithRetry sends an HTTP DELETE request with retry logic.
func DeleteWithRetry(url string, options ...RequestOption) (*Response, []error) {
	return RequestWithRetry(http.MethodDelete, url, options...)
}

// Request executes an HTTP request with the given method, URL, and options.
// It returns a custom Response (assumed to be defined elsewhere) or an error.
func Request(method, url string, options ...RequestOption) (*Response, error) {
	client := http.DefaultClient
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		err = option(req, client)
		if err != nil {
			return nil, err
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return &Response{resp}, nil
}

// Head sends an HTTP HEAD request.
func Head(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodHead, url, options...)
}

// Options sends an HTTP OPTIONS request.
func Options(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodOptions, url, options...)
}

// Get sends an HTTP GET request.
func Get(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodGet, url, options...)
}

// Post sends an HTTP POST request.
func Post(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodPost, url, options...)
}

// Put sends an HTTP PUT request.
func Put(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodPut, url, options...)
}

// Patch sends an HTTP PATCH request.
func Patch(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodPatch, url, options...)
}

// Delete sends an HTTP DELETE request.
func Delete(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodDelete, url, options...)
}
