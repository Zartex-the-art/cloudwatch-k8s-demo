package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestMetricsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	metricsHandler(w, req)
	if !strings.Contains(w.Body.String(), "app_uptime_seconds") {
		t.Errorf("missing app_uptime_seconds")
	}
	if !strings.Contains(w.Body.String(), "app_requests_total") {
		t.Errorf("missing app_requests_total")
	}
}
