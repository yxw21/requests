package requests

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

// Response wraps http.Response and provides helper methods.
// This implementation includes both the default processing methods and raw methods.
type Response struct {
	*http.Response
}

// Text returns the response body as a string using Go's default processing,
// which will use the built-in automatic gzip decoding (if enabled).
func (r *Response) Text() (string, error) {
	defer r.Response.Body.Close()
	body, err := io.ReadAll(r.Response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Bytes returns the response body as a byte slice using Go's default processing.
func (r *Response) Bytes() ([]byte, error) {
	defer r.Response.Body.Close()
	return io.ReadAll(r.Response.Body)
}

// JSON decodes the response body into the provided variable v using Go's default processing.
func (r *Response) JSON(v any) error {
	defer r.Response.Body.Close()
	return json.NewDecoder(r.Response.Body).Decode(v)
}

// XML decodes the response body into the provided variable v using Go's default processing.
func (r *Response) XML(v any) error {
	defer r.Response.Body.Close()
	return xml.NewDecoder(r.Response.Body).Decode(v)
}

// RawText returns the raw response body as a string.
// This should be used when the user disables automatic gzip processing.
func (r *Response) RawText() (string, error) {
	defer r.Response.Body.Close()
	data, err := io.ReadAll(r.Response.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RawBytes returns the raw response body as a byte slice.
// This should be used when the user disables automatic gzip processing.
func (r *Response) RawBytes() ([]byte, error) {
	defer r.Response.Body.Close()
	return io.ReadAll(r.Response.Body)
}

// SafeGzipDecode attempts to decompress the provided data using gzip.
// If decompression fails (e.g. due to an invalid gzip header), it returns the original data.
// This helper allows users to first try gzip decoding on a raw response and fall back if necessary.
func SafeGzipDecode(data []byte) []byte {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		// Decoding failed: return original data.
		return data
	}
	defer gzReader.Close()
	decoded, err := io.ReadAll(gzReader)
	if err != nil {
		// If an error occurs during decoding, return original data.
		return data
	}
	return decoded
}

// RequestCookies parses cookies from the original request header.
func (r *Response) RequestCookies() []*http.Cookie {
	var cookies []*http.Cookie
	cookieHeader := r.Request.Header.Get("Cookie")
	if cookieHeader == "" {
		return cookies
	}
	// Split the cookie header by semicolon and parse each part.
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eqIndex := strings.Index(part, "=")
		if eqIndex == -1 {
			continue
		}
		name := strings.TrimSpace(part[:eqIndex])
		value := ""
		if eqIndex < len(part)-1 {
			value = strings.TrimSpace(part[eqIndex+1:])
		}
		cookies = append(cookies, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	return cookies
}

// ResponseCookies returns cookies from the response by using a cookie jar.
// This method simulates cookie extraction from the response.
func (r *Response) ResponseCookies() ([]*http.Cookie, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", r.Request.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	jar.SetCookies(req.URL, r.Cookies())
	return jar.Cookies(req.URL), nil
}
