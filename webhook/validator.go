package main

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

// validatePod inspects an AdmissionRequest and enforces the rule that every Pod
// must carry a "team" label.
//
// AdmissionRequest context:
//   - UID: opaque identifier the API server assigned to this admission call; we
//     must echo it back in our response so the server can match request/response.
//   - Object.Raw: the JSON-encoded object being admitted (a Pod in our case).
//     The API server sends the raw bytes rather than a typed struct so that the
//     webhook works across object versions without needing version-specific code.
//   - Resource / SubResource / Operation: describe what is happening (e.g. CREATE
//     a Pod) — useful if one handler covers multiple resource types.
//
// We decode Object.Raw ourselves into a corev1.Pod so we can inspect its labels.
func validatePod(admReq admissionv1.AdmissionRequest) (allowed bool, message string) {
	// Guard against an empty or nil Raw payload. This can happen if the API
	// server sends a DELETE review where the object has already been removed, or
	// if a malformed request arrives. Attempting to unmarshal nil bytes would
	// succeed with a zero-value Pod, silently bypassing the label check.
	if len(admReq.Object.Raw) == 0 {
		return false, "admission request contains no object data"
	}

	// Decode the raw pod bytes into a typed struct so we can read its metadata.
	var pod corev1.Pod
	if err := json.Unmarshal(admReq.Object.Raw, &pod); err != nil {
		return false, fmt.Sprintf("could not decode pod object: %v", err)
	}

	// Enforce the labelling policy: every Pod must have a "team" label so that
	// ownership is always traceable in a multi-team cluster.
	if _, ok := pod.Labels["team"]; !ok {
		return false, "pod must have a 'team' label"
	}

	return true, ""
}
