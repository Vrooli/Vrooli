# CLI Commands

This document captures the canonical Swarm Manager CLI flows for backlog import and initiative management.

## Backlog Create

```bash
swarm-manager backlog create --data '{
  "kind":"idea",
  "name":"my-feature",
  "title":"My Feature",
  "acceptance_allow":["scenarios/swarm-manager/**"]
}'
```

Rules:
- input JSON is decoded strictly
- unknown fields fail fast
- do not send `scope`

## Backlog Batch Create

Preview first:

```bash
cat > /tmp/batch-items.json <<'EOF'
{
  "items": [
    {
      "kind": "research",
      "name": "desktop-release-control-plane-audit",
      "title": "Audit desktop release control plane",
      "initiative": "desktop-release-governance",
      "acceptance_allow": ["scenarios/deployment-manager/**"]
    }
  ],
  "initiatives": [
    {
      "name": "desktop-release-governance",
      "title": "Desktop Release Governance",
      "status": "active"
    }
  ]
}
EOF

swarm-manager backlog batch-create --file /tmp/batch-items.json --preview
```

Create after review:

```bash
swarm-manager backlog batch-create --file /tmp/batch-items.json
```

Notes:
- `initiative` is carried per item inside the JSON file
- initiative metadata lives in the top-level `initiatives` array
- there is no `--initiative` flag for batch create

## Initiatives

Create:

```bash
swarm-manager initiatives create --data '{
  "name":"desktop-release-governance",
  "title":"Desktop Release Governance",
  "description":"Shared release-control and desktop delivery work.",
  "status":"active"
}'
```

Update partially:

```bash
swarm-manager initiatives update --name desktop-release-governance --data '{
  "title":"Desktop Release Governance"
}'
```

## Execution Create

If `--mode` is omitted, the CLI resolves it from `GET /api/v1/settings`.

```bash
swarm-manager execution create --kind idea --name my-feature
```
