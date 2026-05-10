# Project: k8s-webhooks (Stage 2 — Build)

## Goal
Build a Kubernetes Validating Admission Webhook HTTP server in Go.
This is a **learning project** — code must be clean, idiomatic, and heavily commented
so a reader can understand every decision.

## What to build

Three Go files under `webhook/`:

### webhook/validator.go
- Package `main`
- Function `validatePod(admReq admissionv1.AdmissionRequest) (allowed bool, message string)`
- Rule: Pod must have a label with key `team`
- If missing: return `false`, message `"pod must have a 'team' label"`
- If present: return `true`, `""`
- Add a comment block explaining what AdmissionRequest contains and why we inspect `.Object.Raw`

### webhook/handler.go
- Package `main`
- HTTP handler func `handleValidate(w http.ResponseWriter, r *http.Request)`
- Decode `AdmissionReview` from request body (JSON)
- Call `validatePod` with `review.Request`
- Build `AdmissionReview` response — uid MUST echo `review.Request.UID`
- Write JSON response with correct Content-Type (`application/json`)
- Handle errors: bad decode → HTTP 400, bad pod decode → deny with message
- Comment every step: why uid must echo, what allowed/denied response looks like

### webhook/main.go
- Package `main`
- Read cert/key paths from env vars `TLS_CERT` and `TLS_KEY` (default: `certs/tls.crt`, `certs/tls.key`)
- Start HTTPS server on `:8443`
- Register route `POST /validate` → `handleValidate`
- Log startup info (port, cert path)
- Comment why TLS is required for k8s webhooks

## Go module
- Module name: `github.com/local/k8s-webhooks`
- Dependencies: only `k8s.io/api` and `k8s.io/apimachinery` (for AdmissionReview types)
- Run `go mod tidy` after writing files

## Also create

### certs/gen-certs.sh
- Bash script using openssl
- Step 1: Generate CA key + self-signed CA cert
- Step 2: Generate server key + CSR
- Step 3: Create extfile with SAN `DNS:webhook-svc.default.svc,DNS:localhost,IP:127.0.0.1`
- Step 4: Sign server cert with CA
- Output: `certs/ca.crt`, `certs/ca.key`, `certs/tls.crt`, `certs/tls.key`
- Make it executable
- Add comments explaining each step and why SAN is required

### Makefile
Targets:
- `build`  — `go build -o bin/webhook ./webhook/`
- `certs`  — `bash certs/gen-certs.sh`
- `run`    — `TLS_CERT=certs/tls.crt TLS_KEY=certs/tls.key ./bin/webhook`
- `test`   — `go test ./webhook/...`

## Code standards
- Every exported and non-trivial unexported func has a comment
- No magic numbers or unexplained values
- Errors are always handled — never `_`
- Use `log.Printf` for logging, not `fmt.Println`
- Standard library only except for k8s types

## Verification
After writing all files:
1. Run `go build ./webhook/` — must succeed with zero errors
2. Run `go vet ./webhook/` — must pass clean
3. Report a file tree summary of what was created
