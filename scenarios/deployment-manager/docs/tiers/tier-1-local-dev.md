# Tier 1 · Local / Developer Stack

Tier 1 is our current reality: every scenario runs inside a single Vrooli installation (workstation, homelab, or developer VPS) and app-monitor exposes them through a Cloudflare Zero Trust tunnel. Treat this as the "reference deployment" that everything else must emulate.

## Current State

- Scenarios are started via `make start`/`vrooli scenario start` and serve **production bundles** so the experience matches shipping bits.
- app-monitor proxies every scenario, providing consistent URLs and auth.
- A Cloudflare tunnel maps the app-monitor proxy to the public internet so we can use phones/laptops anywhere.
- Secrets live in the local secrets-manager (filesystem + Vault) using dev credentials.

## Strengths

- ✅ Zero packaging effort — instant iteration.
- ✅ Full access to every shared resource (Ollama, Postgres, Redis, Qdrant, etc.).
- ✅ Ideal for future household/enterprise appliances because it's literally the full stack.

## Limitations

- ❌ "Deployment" equals "keep the workstation on" — no portability.
- ❌ Secrets are dev-only; we cannot hand this environment to customers.
- ❌ Resource requirements assume desktop/server hardware.

## Documentation Links

- [Production bundles](../../scenarios/PRODUCTION_BUNDLES.md) — why tier 1 uses prod builds.
- [Cloudflare tunnel setup](../providers/digitalocean.md#cloudflare-tunnel-quick-reference) — temporary home until a dedicated guide exists.

## Roadmap

1. Formalize Tier 1 as a deployment mode inside `service.json` (`deployment.platforms.local_dev`).
2. Expose app-monitor + Cloudflare configuration as a reusable `scenario start` target (`vrooli scenario start picker-wheel --target local-dev`).
3. Feed Tier 1 metrics into deployment-manager as the baseline fitness score.
