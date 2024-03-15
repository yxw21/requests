package requests

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
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

type Map map[string]any

type Files []FileField

type FileField struct {
	FieldName string
	FileName  string
	File      *os.File
	FilePath  string
}

func closeFile(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

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

type RequestOption func(*http.Request, *http.Client) error

func WithStringBody(body string) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		req.Body = io.NopCloser(strings.NewReader(body))
		req.ContentLength = int64(len([]byte(body)))
		req.Header.Set("Content-Type", "text/plain")
		return nil
	}
}

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

func WithMultipartFiles(files Files, otherFields Map) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		var err error
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
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
		for key, value := range otherFields {
			if err = writer.WriteField(key, fmt.Sprintf("%v", value)); err != nil {
				return err
			}
		}
		if err := writer.Close(); err != nil {
			return err
		}
		req.Body = io.NopCloser(body)
		req.ContentLength = int64(body.Len())
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return nil
	}
}

func WithHeaders(headers Map) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		for key, value := range headers {
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

func WithTimeout(timeout time.Duration) RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		client.Timeout = timeout
		return nil
	}
}

func WithDisableRedirect() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

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

func WithHTTP2() RequestOption {
	return func(_ *http.Request, client *http.Client) error {
		tr := ensureTransport(client)
		tr.ForceAttemptHTTP2 = true
		return nil
	}
}

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

func WithBasicAuth(username, password string) RequestOption {
	return func(req *http.Request, _ *http.Client) error {
		req.SetBasicAuth(username, password)
		return nil
	}
}

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

func Head(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodHead, url, options...)
}

func Options(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodOptions, url, options...)
}

func Get(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodGet, url, options...)
}

func Post(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodPost, url, options...)
}

func Put(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodPut, url, options...)
}

func Patch(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodPatch, url, options...)
}

func Delete(url string, options ...RequestOption) (*Response, error) {
	return Request(http.MethodDelete, url, options...)
}
