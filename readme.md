# Go Requests: Lightweight HTTP Request Library

`requests` is a lightweight Go library designed to simplify the process of making HTTP requests. It provides a concise set of interfaces for common HTTP operations, including `OPTIONS`, `HEAD`, `GET`, `POST`, `PUT`, `PATCH`, `DELETE` requests, along with support for `Text`, `JSON`, `XML`, form data formats, file uploads, and convenient handling of responses.

## Features

- **Simple API Design**: Configuring requests is made easy with the Functional Options Pattern, making the code easy to read and maintain.
- **Flexible Request and Response Handling**: Supports multiple request body formats, including `Text`, `JSON`, `XML`, form data, and multipart file uploads. Responses can be directly read as text, byte sequences, JSON, or XML objects.
- **Advanced HTTP Functionality Support**: Includes features like basic authentication, custom request headers, proxy settings, SSL certificate verification bypass, etc.
- **Automated Cookie and Redirect Handling**: Built-in cookie jar for automatic handling of cookies in requests and responses, with configurable redirect policies.

## Installation

Install using the Go command line tool:

```bash
go get github.com/yxw21/requests
```

Make sure your project path has Go Modules set up correctly.

## Quick Start

Here are a few examples of sending HTTP requests using the `requests` library.

### Sending a GET Request

```go
resp, err := requests.Get("http://httpbin.org/get")
if err != nil {
    fmt.Println("Request failed:", err)
    return
}
text, _ := resp.Text()
fmt.Println("Response text:", text)
```

### Sending a JSON POST Request

```go
resp, err := requests.Post("http://httpbin.org/post",
    requests.WithJSONBody(requests.Map{"key": "value"}),
)
if err != nil {
    fmt.Println("Request failed:", err)
    return
}
jsonResponse := make(requests.Map)
resp.JSON(&jsonResponse)
fmt.Println("JSON response:", jsonResponse)
```

### Uploading a File

```go
files := requests.Files{
    {FieldName: "file", FileName: "file.txt", FilePath: "/path/1.txt"},
}
resp, err := requests.Post("http://httpbin.org/post",
    requests.WithMultipartFiles(files, nil),
)
if err != nil {
    fmt.Println("Request failed:", err)
    return
}
text, _ := resp.Text()
fmt.Println("Response text:", text)
```
