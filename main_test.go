package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateRequestURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "HTTPS", url: "https://example.com/article"},
		{name: "HTTP", url: "http://example.com/article"},
		{name: "Loopback", url: "http://127.0.0.1/admin", wantErr: true},
		{name: "IPv6 loopback", url: "http://[::1]/admin", wantErr: true},
		{name: "Private network", url: "http://10.0.0.1/admin", wantErr: true},
		{name: "Link local", url: "http://169.254.169.254/latest/meta-data", wantErr: true},
		{name: "Credentials", url: "https://user:password@example.com", wantErr: true},
		{name: "Unsupported scheme", url: "file:///etc/passwd", wantErr: true},
		{name: "Missing host", url: "http:///path", wantErr: true},
		{name: "Relative URL", url: "/relative", wantErr: true},
		{name: "Localhost", url: "http://localhost/admin", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequestURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRequestURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestHandleExtractRejectsPrivateTarget(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/url/?url=http://127.0.0.1/admin", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(handleExtract).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("handler returned status %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var result Output
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if result.Success {
		t.Fatal("expected request to be rejected")
	}
}

func TestHealthCheckHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleStatus)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	var response StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if !response.Alive {
		t.Error("expected alive to be true")
	}
}
