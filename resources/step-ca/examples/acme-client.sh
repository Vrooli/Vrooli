#!/bin/bash
# Example: Using Step-CA with ACME clients

# ACME directory URL
ACME_DIR="https://localhost:9010/acme/acme/directory"

# Example 1: Using certbot for certificate enrollment
echo "=== Certbot Example ==="
echo "# Install certbot"
echo "sudo apt-get install certbot"
echo ""
echo "# Request certificate"
echo "certbot certonly \\"
echo "  --server $ACME_DIR \\"
echo "  --standalone \\"
echo "  --non-interactive \\"
echo "  --agree-tos \\"
echo "  --email admin@vrooli.local \\"
echo "  -d service.vrooli.local"
echo ""

# Example 2: Using acme.sh
echo "=== acme.sh Example ==="
echo "# Install acme.sh"
echo "curl https://get.acme.sh | sh"
echo ""
echo "# Request certificate"
echo "acme.sh --issue \\"
echo "  --server $ACME_DIR \\"
echo "  -d service.vrooli.local \\"
echo "  --standalone"
echo ""

# Example 3: Using Step CLI directly
echo "=== Step CLI Example ==="
echo "# Install step CLI"
echo "wget https://dl.step.sm/gh-release/cli/latest/step_linux_amd64.tar.gz"
echo "tar -xf step_linux_amd64.tar.gz"
echo "sudo mv step_*/bin/step /usr/local/bin/"
echo ""
echo "# Bootstrap with root certificate"
echo "step ca bootstrap --ca-url https://localhost:9010 \\"
echo "  --fingerprint \$(step certificate fingerprint ~/Vrooli/data/step-ca/certs/root_ca.crt)"
echo ""
echo "# Request certificate with token"
echo "step ca certificate service.vrooli.local service.crt service.key \\"
echo "  --provisioner admin \\"
echo "  --not-after 24h"
echo ""

# Example 4: Automatic renewal with systemd
echo "=== Automatic Renewal with systemd ==="
cat <<'EOF'
# Create renewal service: /etc/systemd/system/cert-renewal.service
[Unit]
Description=Certificate Renewal
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet
User=root

[Install]
WantedBy=multi-user.target

# Create renewal timer: /etc/systemd/system/cert-renewal.timer
[Unit]
Description=Certificate Renewal Timer
Requires=cert-renewal.service

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target

# Enable automatic renewal
sudo systemctl enable cert-renewal.timer
sudo systemctl start cert-renewal.timer
EOF

echo ""
echo "Note: Replace 'service.vrooli.local' with your actual domain name"
echo "Note: For production, distribute the root certificate from:"
echo "      ~/Vrooli/data/step-ca/certs/root_ca.crt"