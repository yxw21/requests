package requests

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

type Response struct {
	*http.Response
}

func (r *Response) Text() (string, error) {
	defer r.Response.Body.Close()
	body, err := io.ReadAll(r.Response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (r *Response) Bytes() ([]byte, error) {
	defer r.Response.Body.Close()
	return io.ReadAll(r.Response.Body)
}

func (r *Response) JSON(v any) error {
	defer r.Response.Body.Close()
	return json.NewDecoder(r.Response.Body).Decode(v)
}

func (r *Response) XML(v any) error {
	defer r.Response.Body.Close()
	return xml.NewDecoder(r.Response.Body).Decode(v)
}

func (r *Response) RequestCookies() []*http.Cookie {
	var cookies []*http.Cookie
	cookieHeader := r.Request.Header.Get("Cookie")
	if cookieHeader == "" {
		return cookies
	}
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
	cookies := jar.Cookies(req.URL)
	return cookies, nil
}
