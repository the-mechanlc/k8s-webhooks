#!/usr/bin/env bash
# gen-certs.sh — Generate a self-signed CA and a server TLS certificate for
# local development and in-cluster testing of the admission webhook.
#
# Output files (all written to the directory containing this script):
#   ca.key    — CA private key
#   ca.crt    — Self-signed CA certificate
#   tls.key   — Server private key
#   tls.crt   — Server certificate signed by the CA above
#
# The CA cert (ca.crt) must be base64-encoded and placed in the
# ValidatingWebhookConfiguration's .webhooks[].clientConfig.caBundle field so
# that the Kubernetes API server can verify the server certificate when it calls
# the webhook.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ---------------------------------------------------------------------------
# Step 1: Generate the Certificate Authority (CA)
#
# We create our own CA rather than using a well-known public CA because:
#   a) The webhook runs on a private in-cluster DNS name that no public CA
#      will sign.
#   b) Kubernetes lets us supply our own CA bundle in the webhook config, so
#      a self-signed CA is perfectly valid.
# ---------------------------------------------------------------------------
echo "==> Generating CA key and self-signed certificate..."
openssl genrsa -out ca.key 2048

openssl req -new -x509 \
  -key ca.key \
  -out ca.crt \
  -days 365 \
  -subj "/CN=webhook-ca"
# SECURITY: ca.key is the root of trust for this webhook — keep it secret.
# Anyone who holds ca.key can issue certificates that the API server will trust.

# ---------------------------------------------------------------------------
# Step 2: Generate the server private key and a Certificate Signing Request
#
# The CN here is less important than the SAN (see Step 3), but we use the
# primary service DNS name for clarity.
# ---------------------------------------------------------------------------
echo "==> Generating server key and CSR..."
openssl genrsa -out tls.key 2048

openssl req -new \
  -key tls.key \
  -out tls.csr \
  -subj "/CN=webhook-svc.default.svc"

# ---------------------------------------------------------------------------
# Step 3: Create a SAN extension file
#
# Why Subject Alternative Names (SAN) are required:
#   Modern TLS clients (including Go's crypto/tls) reject certificates that
#   do not carry a SAN extension. The Common Name (CN) field is no longer
#   used for hostname verification (RFC 2818 / Go 1.15+).
#
#   We include:
#     DNS:webhook-svc.default.svc  — the in-cluster service FQDN the API
#                                    server uses when the webhook is deployed
#     DNS:localhost                — for local testing with `make run`
#     IP:127.0.0.1                 — for curl / integration tests hitting
#                                    127.0.0.1 directly
# ---------------------------------------------------------------------------
echo "==> Writing SAN extension file..."
cat > san.ext <<EOF
subjectAltName = DNS:webhook-svc.default.svc,DNS:localhost,IP:127.0.0.1
EOF

# ---------------------------------------------------------------------------
# Step 4: Sign the server certificate with our CA
#
# -CA / -CAkey       point to the CA we created in Step 1
# -CAcreateserial    writes a serial number file (ca.srl) on first use
# -extfile san.ext   attaches the SAN extension from Step 3
# ---------------------------------------------------------------------------
echo "==> Signing server certificate with CA..."
openssl x509 -req \
  -in tls.csr \
  -CA ca.crt \
  -CAkey ca.key \
  -CAcreateserial \
  -out tls.crt \
  -days 365 \
  -extfile san.ext

# Clean up the CSR and extension file — only the final certs are needed.
rm -f tls.csr san.ext

echo ""
echo "Done. Files written to ${SCRIPT_DIR}:"
ls -lh ca.crt ca.key tls.crt tls.key
echo ""
echo "To register this CA with a ValidatingWebhookConfiguration:"
echo "  caBundle: \$(base64 -w 0 ${SCRIPT_DIR}/ca.crt)"
