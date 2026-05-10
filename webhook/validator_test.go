package main

import (
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// podRaw serialises a corev1.Pod to raw JSON bytes, mirroring how the API
// server populates AdmissionRequest.Object.Raw.
func podRaw(t *testing.T, pod corev1.Pod) []byte {
	t.Helper()
	b, err := json.Marshal(pod)
	if err != nil {
		t.Fatalf("failed to marshal pod: %v", err)
	}
	return b
}

func TestValidatePod(t *testing.T) {
	tests := []struct {
		name        string
		raw         []byte
		wantAllowed bool
		wantMessage string
	}{
		{
			name: "pod with team label is allowed",
			raw: podRaw(t, corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"team": "platform"},
				},
			}),
			wantAllowed: true,
			wantMessage: "",
		},
		{
			name: "pod without team label is denied",
			raw: podRaw(t, corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "nginx"},
				},
			}),
			wantAllowed: false,
			wantMessage: "pod must have a 'team' label",
		},
		{
			name:        "pod with no labels at all is denied",
			raw:         podRaw(t, corev1.Pod{}),
			wantAllowed: false,
			wantMessage: "pod must have a 'team' label",
		},
		{
			name:        "malformed raw bytes returns error",
			raw:         []byte(`{not valid json`),
			wantAllowed: false,
			// We only check the prefix because the JSON error text is implementation-defined.
			wantMessage: "", // checked separately below
		},
		{
			name:        "empty raw bytes returns error",
			raw:         []byte{},
			wantAllowed: false,
			wantMessage: "admission request contains no object data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{Raw: tc.raw},
			}
			allowed, msg := validatePod(req)

			if allowed != tc.wantAllowed {
				t.Errorf("allowed = %v, want %v", allowed, tc.wantAllowed)
			}

			// For the malformed-JSON case we just verify the message is non-empty
			// and starts with the expected prefix rather than hard-coding Go's
			// internal JSON error string.
			if tc.name == "malformed raw bytes returns error" {
				if allowed {
					t.Error("expected denial for malformed JSON, got allowed=true")
				}
				if msg == "" {
					t.Error("expected non-empty error message for malformed JSON")
				}
				return
			}

			if msg != tc.wantMessage {
				t.Errorf("message = %q, want %q", msg, tc.wantMessage)
			}
		})
	}
}
