package requests

import (
	"fmt"
	"testing"
	"time"
)

func TestAllMethods(t *testing.T) {
	serverURL := "http://localhost:8080"
	t.Run("OPTIONS", func(t *testing.T) {
		resp, err := Options(serverURL+"/?query=options", WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("OPTIONS request failed: %v", err)
		}
		fmt.Println(resp.StatusCode)
	})
	t.Run("HEAD", func(t *testing.T) {
		resp, err := Head(serverURL+"/?query=head", WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("HEAD request failed: %v", err)
		}
		fmt.Println(resp.StatusCode)
	})
	t.Run("GET", func(t *testing.T) {
		resp, err := Get(serverURL+"/?query=get", WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		var result map[string]any
		if err := resp.JSON(&result); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}
		fmt.Println(result)
	})
	t.Run("POST", func(t *testing.T) {
		data := map[string]any{
			"key": "value",
		}
		resp, err := Post(serverURL+"/?query=post", WithJSONBody(data), WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		var result map[string]any
		if err := resp.JSON(&result); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}
		fmt.Println(result)
	})
	t.Run("Upload", func(t *testing.T) {
		resp, err := Post(serverURL+"/upload", WithMultipartFiles(Files{
			{FieldName: "file", FileName: "11.txt", FilePath: "/tmp/11.txt"},
		}, Map{
			"upload": "true",
		}), WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("Upload request failed: %v", err)
		}
		var result map[string]any
		if err := resp.JSON(&result); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}
		fmt.Println(result)
	})
}
