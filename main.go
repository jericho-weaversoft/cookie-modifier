// main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Config holds the plugin configuration
type Config struct {
	SourceCookieName string `json:"sourceCookieName,omitempty"`
	TargetCookieName string `json:"targetCookieName,omitempty"`
	UseDynamicDomain bool   `json:"useDynamicDomain,omitempty"`
	Secure           bool   `json:"secure,omitempty"`
	HttpOnly         bool   `json:"httpOnly,omitempty"`
	SameSite         string `json:"sameSite,omitempty"`
	Path             string `json:"path,omitempty"`
	Debug            bool   `json:"debug,omitempty"`
}

// CreateConfig creates the default plugin configuration
func CreateConfig() *Config {
	return &Config{
		SourceCookieName: "flowise_token",
		TargetCookieName: "simple_token",
		UseDynamicDomain: true,
		Secure:           false,
		HttpOnly:         false,
		SameSite:         "Lax",
		Path:             "/",
		Debug:            false,
	}
}

// CookieModifier holds the plugin instance
type CookieModifier struct {
	next   http.Handler
	config *Config
	name   string
}

// New creates a new plugin instance
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	// Validate configuration
	if config.SourceCookieName == "" {
		return nil, fmt.Errorf("sourceCookieName cannot be empty")
	}
	if config.TargetCookieName == "" {
		return nil, fmt.Errorf("targetCookieName cannot be empty")
	}

	if config.Debug {
		fmt.Printf("[Cookie Modifier] Plugin initialized with config: %+v\n", config)
	}

	return &CookieModifier{
		next:   next,
		config: config,
		name:   name,
	}, nil
}

// ServeHTTP handles the HTTP request
func (cm *CookieModifier) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if cm.config.Debug {
		fmt.Printf("[Cookie Modifier] Processing request to: %s\n", req.URL.String())
	}

	// Transform request cookies
	cm.transformRequestCookies(req)

	// Create a custom response writer to intercept response cookies
	wrappedWriter := &responseWriter{
		ResponseWriter: rw,
		req:            req,
		config:         cm.config,
	}

	// Continue to the next handler
	cm.next.ServeHTTP(wrappedWriter, req)
}

// transformRequestCookies modifies cookies in the incoming request
func (cm *CookieModifier) transformRequestCookies(req *http.Request) {
	cookies := req.Cookies()
	var newCookies []*http.Cookie
	var foundSourceCookie *http.Cookie

	if cm.config.Debug {
		fmt.Printf("[Cookie Modifier] Found %d cookies in request\n", len(cookies))
	}

	// Find the source cookie and collect other cookies
	for _, cookie := range cookies {
		if cookie.Name == cm.config.SourceCookieName {
			foundSourceCookie = cookie
			if cm.config.Debug {
				fmt.Printf("[Cookie Modifier] Found source cookie: %s=%s\n", cookie.Name, cookie.Value)
			}
		} else {
			newCookies = append(newCookies, cookie)
		}
	}

	// If source cookie found, create the transformed cookie
	if foundSourceCookie != nil {
		transformedCookie := &http.Cookie{
			Name:  cm.config.TargetCookieName,
			Value: foundSourceCookie.Value,
			Path:  cm.config.Path,
		}

		// Set domain to the target URL if dynamic domain is enabled
		if cm.config.UseDynamicDomain {
			// Use the Host header to determine the domain
			if req.Host != "" {
				transformedCookie.Domain = req.Host
				if cm.config.Debug {
					fmt.Printf("[Cookie Modifier] Set dynamic domain to: %s\n", req.Host)
				}
			}
		}

		newCookies = append(newCookies, transformedCookie)

		// Rebuild the Cookie header
		var cookieStrings []string
		for _, cookie := range newCookies {
			cookieStrings = append(cookieStrings, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
		req.Header.Set("Cookie", strings.Join(cookieStrings, "; "))

		if cm.config.Debug {
			fmt.Printf("[Cookie Modifier] Transformed cookie: %s -> %s\n", 
				cm.config.SourceCookieName, cm.config.TargetCookieName)
		}
	}
}

// responseWriter wraps http.ResponseWriter to intercept Set-Cookie headers
type responseWriter struct {
	http.ResponseWriter
	req    *http.Request
	config *Config
}

// WriteHeader intercepts response headers to modify Set-Cookie
func (rw *responseWriter) WriteHeader(statusCode int) {
	// Process Set-Cookie headers in the response
	rw.transformResponseCookies()
	rw.ResponseWriter.WriteHeader(statusCode)
}

// transformResponseCookies modifies Set-Cookie headers in the response
func (rw *responseWriter) transformResponseCookies() {
	setCookieHeaders := rw.Header().Values("Set-Cookie")
	if len(setCookieHeaders) == 0 {
		return
	}

	if rw.config.Debug {
		fmt.Printf("[Cookie Modifier] Processing %d Set-Cookie headers\n", len(setCookieHeaders))
	}

	var newSetCookieHeaders []string

	for _, setCookieHeader := range setCookieHeaders {
		// Check if this Set-Cookie header contains our source cookie
		if strings.Contains(setCookieHeader, rw.config.SourceCookieName+"=") {
			// Transform this cookie
			transformedHeader := strings.Replace(setCookieHeader,
				rw.config.SourceCookieName+"=",
				rw.config.TargetCookieName+"=", 1)

			// Add domain if dynamic domain is enabled and not already present
			if rw.config.UseDynamicDomain && !strings.Contains(transformedHeader, "Domain=") {
				transformedHeader += fmt.Sprintf("; Domain=%s", rw.req.Host)
			}

			// Add path if not already present
			if rw.config.Path != "/" && !strings.Contains(transformedHeader, "Path=") {
				transformedHeader += fmt.Sprintf("; Path=%s", rw.config.Path)
			}

			// Add security attributes
			if rw.config.Secure && !strings.Contains(transformedHeader, "Secure") {
				transformedHeader += "; Secure"
			}
			if rw.config.HttpOnly && !strings.Contains(transformedHeader, "HttpOnly") {
				transformedHeader += "; HttpOnly"
			}
			if rw.config.SameSite != "" && !strings.Contains(transformedHeader, "SameSite=") {
				transformedHeader += fmt.Sprintf("; SameSite=%s", rw.config.SameSite)
			}

			newSetCookieHeaders = append(newSetCookieHeaders, transformedHeader)

			if rw.config.Debug {
				fmt.Printf("[Cookie Modifier] Transformed Set-Cookie: %s\n", transformedHeader)
			}
		} else {
			// Keep other cookies as-is
			newSetCookieHeaders = append(newSetCookieHeaders, setCookieHeader)
		}
	}

	// Replace Set-Cookie headers
	rw.Header().Del("Set-Cookie")
	for _, header := range newSetCookieHeaders {
		rw.Header().Add("Set-Cookie", header)
	}
}