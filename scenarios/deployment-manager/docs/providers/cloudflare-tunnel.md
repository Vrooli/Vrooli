# Provider Notes · Cloudflare Tunnel

> **Status:** Active Tier 1 practice. Use this guide to expose app-monitor (and therefore every running scenario) to the internet without opening firewall ports.

Tier 1 deployments rely on a single Vrooli stack plus app-monitor. Cloudflare tunnels give us secure remote access from phones and laptops without deploying extra infrastructure.

## Prerequisites

- Cloudflare account with a domain you control
- Access to the machine running Vrooli + app-monitor (local workstation or VPS)
- `cloudflared` binary installed (instructions below)

## Setup Steps

1. **Install cloudflared**
   ```bash
   curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb -o cloudflared.deb
   sudo dpkg -i cloudflared.deb
   ```

2. **Authenticate and create a tunnel**
   ```bash
   cloudflared login            # Browser prompts for Cloudflare auth
   cloudflared tunnel create vrooli-dev
   cloudflared tunnel route dns vrooli-dev apps.<your-domain>
   ```

3. **Configure ingress rules** (proxy app-monitor which already front-loads every scenario)
   ```yaml
   # ~/.cloudflared/config.yml
   tunnel: vrooli-dev
   credentials-file: /home/$USER/.cloudflared/vrooli-dev.json
   ingress:
     - hostname: apps.<your-domain>
       service: http://localhost:3443   # app-monitor proxy URL
     - service: http_status:404
   ```

4. **Run as a service**
   ```bash
   sudo cloudflared service install
   sudo systemctl enable --now cloudflared
   ```

5. **Optional: Multiple hostnames**
   You can add additional ingress entries for specific scenarios (`scenario-name.<domain>`) that point to app-monitor routes if needed.

## Operational Notes

- Treat this as the supported remote-access method until Tier 2+ packaging exists.
- Rotate tunnel credentials if the VPS is rebuilt.
- Document any customer-specific hostname mappings in `docs/deployment/examples/` so deployment-manager can automate them later.

## Related Docs

- [Tier 1 Local / Developer Stack](../tiers/tier-1-local-dev.md)
- [Deployment Hub](../README.md)
