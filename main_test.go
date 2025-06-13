// main_test.go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCookieModifier_BasicTransformation(t *testing.T) {
	config := CreateConfig()
	config.Debug = true

	// Mock next handler
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check if the cookie was transformed
		cookies := req.Cookies()
		var foundTarget bool
		var foundSource bool

		for _, cookie := range cookies {
			if cookie.Name == "simple_token" {
				foundTarget = true
				if cookie.Value != "test-token-value" {
					t.Errorf("Expected cookie value 'test-token-value', got '%s'", cookie.Value)
				}
			}
			if cookie.Name == "flowise_token" {
				foundSource = true
			}
		}

		if !foundTarget {
			t.Error("Target cookie 'simple_token' not found")
		}
		if foundSource {
			t.Error("Source cookie 'flowise_token' should have been removed")
		}

		rw.WriteHeader(http.StatusOK)
	})

	// Create plugin
	plugin, err := New(context.Background(), next, config, "test-cookie-modifier")
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with source cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "flowise_token",
		Value: "test-token-value",
	})
	req.Host = "example.com"

	rw := httptest.NewRecorder()

	// Execute
	plugin.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.Code)
	}
}

func TestCookieModifier_ResponseTransformation(t *testing.T) {
	config := CreateConfig()
	config.Debug = true

	// Mock next handler that sets a cookie
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Set a cookie in the response
		http.SetCookie(rw, &http.Cookie{
			Name:  "flowise_token",
			Value: "response-token-value",
			Path:  "/",
		})
		rw.WriteHeader(http.StatusOK)
	})

	// Create plugin
	plugin, err := New(context.Background(), next, config, "test-cookie-modifier")
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "example.com"

	rw := httptest.NewRecorder()

	// Execute
	plugin.ServeHTTP(rw, req)

	// Check Set-Cookie header
	setCookieHeaders := rw.Header().Values("Set-Cookie")
	if len(setCookieHeaders) == 0 {
		t.Fatal("No Set-Cookie headers found")
	}

	found := false
	for _, header := range setCookieHeaders {
		if strings.Contains(header, "simple_token=response-token-value") {
			found = true
			if !strings.Contains(header, "Domain=example.com") {
				t.Error("Expected Domain=example.com in Set-Cookie header")
			}
		}
		if strings.Contains(header, "flowise_token=") {
			t.Error("Source cookie should not be present in response")
		}
	}

	if !found {
		t.Error("Transformed cookie not found in Set-Cookie headers")
	}
}

func TestCookieModifier_NoCookieTransformation(t *testing.T) {
	config := CreateConfig()

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Should pass through without modification
		cookies := req.Cookies()
		if len(cookies) != 1 {
			t.Errorf("Expected 1 cookie, got %d", len(cookies))
		}
		if cookies[0].Name != "other_cookie" {
			t.Errorf("Expected 'other_cookie', got '%s'", cookies[0].Name)
		}
		rw.WriteHeader(http.StatusOK)
	})

	plugin, err := New(context.Background(), next, config, "test-cookie-modifier")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "other_cookie",
		Value: "other_value",
	})

	rw := httptest.NewRecorder()
	plugin.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.Code)
	}
}

func TestCookieModifier_InvalidConfig(t *testing.T) {
	config := &Config{
		SourceCookieName: "", // Invalid - empty
		TargetCookieName: "simple_token",
	}

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := New(context.Background(), next, config, "test-cookie-modifier")
	if err == nil {
		t.Error("Expected error for empty sourceCookieName")
	}
}