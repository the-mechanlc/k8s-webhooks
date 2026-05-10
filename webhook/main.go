package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
)

// envOrDefault returns the value of the environment variable named by key, or
// fallback if the variable is not set.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// main is the entry point. It reads TLS configuration from environment
// variables, registers the validation route, and starts the HTTPS server.
//
// Why TLS is required for Kubernetes webhooks:
// The Kubernetes API server only calls webhook endpoints over HTTPS. It verifies
// the server certificate against a CA bundle that must be provided in the
// ValidatingWebhookConfiguration manifest. Without TLS the API server will
// refuse to call the webhook at all, and the relevant admission controller will
// either block object creation (Fail policy) or skip the webhook (Ignore policy).
func main() {
	// Allow operators to override cert/key paths via environment variables so
	// the binary works in any deployment without recompilation.
	certFile := envOrDefault("TLS_CERT", "certs/tls.crt")
	keyFile := envOrDefault("TLS_KEY", "certs/tls.key")

	// Use a named ServeMux instead of the global default so we don't
	// accidentally inherit any routes registered by imported packages.
	mux := http.NewServeMux()

	// POST /validate — admission review endpoint called by the API server.
	mux.HandleFunc("/validate", handleValidate)

	// GET /healthz — liveness probe endpoint. Returns 200 OK with no body.
	// Kubernetes (or any load balancer) can hit this without needing TLS client
	// auth, confirming the process is alive and the server is accepting conns.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Enforce a minimum TLS version of 1.2. TLS 1.0 and 1.1 have known
	// weaknesses (POODLE, BEAST) and are disabled by default in modern
	// Kubernetes API servers; explicitly requiring 1.2+ makes the policy clear.
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	srv := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsCfg,
	}

	log.Printf("starting webhook server on :8443 (cert=%s, key=%s)", certFile, keyFile)

	// ListenAndServeTLS blocks until the server exits. Any startup error
	// (e.g. cert file not found, port already in use) is fatal.
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
