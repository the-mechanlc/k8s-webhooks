package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleValidate is the HTTP handler for POST /validate.
//
// The Kubernetes API server sends a JSON-encoded AdmissionReview in the request
// body, waits for us to respond with another AdmissionReview that contains our
// allow/deny decision, then either admits or rejects the object.
//
// Response shape:
//
//	{
//	  "apiVersion": "admission.k8s.io/v1",
//	  "kind":       "AdmissionReview",
//	  "response": {
//	    "uid":     "<must match request uid>",
//	    "allowed": true | false,
//	    "status":  { "message": "..." }   // only populated on deny
//	  }
//	}
func handleValidate(w http.ResponseWriter, r *http.Request) {
	// Only accept POST — the Kubernetes API server always uses POST for webhooks.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cap the request body at 1 MiB to prevent memory exhaustion from
	// oversized or malformed payloads. A legitimate AdmissionReview is tiny
	// (a few kilobytes at most).
	const maxBodyBytes = 1 << 20 // 1 MiB
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		log.Printf("error reading request body: %v", err)
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	// Decode the incoming AdmissionReview. A malformed payload is a client
	// error, so we respond with 400 rather than a deny decision.
	var review admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &review); err != nil {
		log.Printf("error decoding AdmissionReview: %v", err)
		http.Error(w, "could not decode AdmissionReview", http.StatusBadRequest)
		return
	}

	// A nil Request means the API server sent an AdmissionReview shell with no
	// inner request — this should never happen in practice but we guard
	// defensively to avoid a nil-pointer panic on the dereference below.
	if review.Request == nil {
		log.Printf("AdmissionReview arrived with nil Request")
		http.Error(w, "AdmissionReview missing Request", http.StatusBadRequest)
		return
	}

	// Validate the pod contained in the admission request.
	allowed, message := validatePod(*review.Request)
	if !allowed {
		log.Printf("denying pod (uid=%s): %s", review.Request.UID, message)
	}

	// Build the response AdmissionReview.
	//
	// The UID MUST be an exact echo of review.Request.UID. The API server uses
	// it to correlate the async response back to the original request; if it
	// doesn't match the request is treated as failed.
	resp := admissionv1.AdmissionReview{
		// TypeMeta must be set so the API server recognises the response kind.
		TypeMeta: review.TypeMeta,
		Response: &admissionv1.AdmissionResponse{
			UID:     review.Request.UID, // echo back — required by the protocol
			Allowed: allowed,
		},
	}

	// Only populate Status on a denial; leaving it nil on allow is cleaner.
	if !allowed {
		resp.Response.Result = &metav1.Status{
			Message: message,
		}
	}

	// Encode and write the response. Content-Type must be application/json so
	// the API server's HTTP client parses it correctly.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("error writing response: %v", err)
	}
}
