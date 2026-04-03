package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	startTime    = time.Now()
	requestCount uint64
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/metrics", metricsHandler)
	log.Printf("service=cloudwatch-k8s-demo port=%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(&requestCount, 1)
	log.Printf("method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
	fmt.Fprintln(w, "cloudwatch-k8s-demo — Go HTTP Service")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(&requestCount, 1)
	log.Printf("method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "cloudwatch-k8s-demo",
		"uptime":  time.Since(startTime).String(),
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
	uptime := time.Since(startTime).Seconds()
	count := atomic.LoadUint64(&requestCount)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP app_uptime_seconds Uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE app_uptime_seconds gauge\n")
	fmt.Fprintf(w, "app_uptime_seconds %.0f\n\n", uptime)
	fmt.Fprintf(w, "# HELP app_requests_total Total HTTP requests\n")
	fmt.Fprintf(w, "# TYPE app_requests_total counter\n")
	fmt.Fprintf(w, "app_requests_total %d\n\n", count)
	fmt.Fprintf(w, "# HELP app_info Service info\n")
	fmt.Fprintf(w, "# TYPE app_info gauge\n")
	fmt.Fprintf(w, "app_info{version=\"1.0.0\",service=\"cloudwatch-k8s-demo\"} 1\n")
}
