# K8s Admission Webhooks in Go — Learning Project Plan

## Goal
Build a simple, well-commented Kubernetes Validating Admission Webhook in Go to understand
how the K8s API server admission chain works — from the ground up.

## Stack
- **Go 1.26** (via snap) — standard library only, no controller-runtime
- **k3s** — lightest real Kubernetes, runs natively on Linux (no Docker needed)
- **openssl** — self-signed TLS cert (required by k8s for webhooks)
- **Mermaid** — annotated flow diagrams embedded directly in README (renders natively on GitHub)

---

## Project Structure
```
projects/k8s-webhooks/
├── README.md                  # Concept explainer + 3 embedded Mermaid diagrams
├── webhook/
│   ├── main.go                # HTTP server entrypoint
│   ├── handler.go             # AdmissionReview request/response logic
│   └── validator.go           # Business logic: must have `team` label
├── certs/
│   └── gen-certs.sh           # openssl script to generate TLS cert + key
├── manifests/
│   ├── deployment.yaml        # Deploy webhook server into k3s
│   ├── service.yaml           # ClusterIP service for the webhook
│   └── webhook-config.yaml    # ValidatingWebhookConfiguration
└── Makefile                   # build / cert / deploy / test targets
```

---

## Step-by-Step Plan

### Stage 1 — Understand (diagrams + README)
1. Write `README.md` with 3 embedded Mermaid diagrams:
   - Admission chain flowchart: `kubectl apply` → API server → Auth/AuthZ → Admission Controllers → Validating Webhook → etcd
   - Sequence diagram: HTTP round-trip — AdmissionReview request → webhook → AdmissionResponse
   - TLS setup flowchart: cert chain + caBundle registration
   - Annotate: what fields matter, where allowed/denied is set
2. Include in README:
   - What is an admission webhook (2 types: Validating vs Mutating)
   - What we're building and why
   - Full `ValidatingWebhookConfiguration` YAML example with inline comments

### Stage 2 — Build (Go webhook server)
3. Install Go via snap
4. Install k3s
5. `go mod init github.com/local/k8s-webhooks`
6. Write `webhook/handler.go`
   - Parse `AdmissionReview` from POST body
   - Return `AdmissionResponse` with `allowed: true/false`
   - Heavily commented — explain every field
7. Write `webhook/validator.go`
   - Rule: Pod must have a `team` label
   - Return clear deny message if missing
8. Write `webhook/main.go`
   - HTTPS server on port 8443
   - Mount TLS cert/key from `certs/`
9. Write `certs/gen-certs.sh`
   - Generate CA + server cert signed for the in-cluster service DNS name
   - Output `tls.crt` + `tls.key`

### Stage 3 — Deploy & Test
10. Write `manifests/deployment.yaml` + `manifests/service.yaml`
    - Deploy webhook binary as a pod in k3s
    - Expose as a ClusterIP service
11. Write `manifests/webhook-config.yaml`
    - `ValidatingWebhookConfiguration` pointing to the service
    - `caBundle` from the generated cert
    - Rule: intercept CREATE on Pods
12. `make test` — two test cases:
    - `kubectl apply` a Pod **with** `team` label → should be **accepted** ✅
    - `kubectl apply` a Pod **without** `team` label → should be **rejected** ❌ with message

---

## Files to Create
| File | Purpose |
|------|---------|
| `README.md` | Learning explainer + 3 Mermaid diagrams |
| `webhook/main.go` | TLS HTTP server |
| `webhook/handler.go` | AdmissionReview parsing |
| `webhook/validator.go` | Label validation logic |
| `certs/gen-certs.sh` | TLS cert generation |
| `manifests/*.yaml` | K8s manifests |
| `Makefile` | build/run/test shortcuts |

---

## Concepts Covered
- Admission controller chain in the K8s API server
- `AdmissionReview` / `AdmissionRequest` / `AdmissionResponse` structs
- Why webhooks need TLS (and how caBundle works)
- `ValidatingWebhookConfiguration` — rules, namespaceSelector, failurePolicy
- Difference between Validating and Mutating webhooks
- How to test webhooks locally with k3s

---

## Risks / Notes
- k3s installs a real cluster — takes ~30s to come up, minimal resources
- Self-signed certs require the `caBundle` in the webhook config to match exactly
- `failurePolicy: Fail` vs `Ignore` — we'll use `Ignore` during dev so a broken webhook doesn't lock the cluster
- No external dependencies in Go code — uses only `encoding/json`, `net/http`, `k8s.io/api`
