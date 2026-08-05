# Deployment Manager Architecture

Deployment-manager is the governance plane. Domain policy and persistence are
kept behind repositories; HTTP/Connect handlers translate at the edge. The
evidence plane produces target verdicts, the reach plane supplies bridge
results, and the governance plane joins exact-commit evidence with human
approvals.

```mermaid
flowchart LR
  producer["Ramp producer"] -->|TargetVerdict + EvidenceRef| ingest["EvidenceService"]
  bridge["vrooli-bridge"] -->|host/physical verdict| ingest
  ingest --> gate["Release gate"]
  approval["Human approval"] --> gate
  gate --> decision["allow / refuse with named reason"]
  producer -->|producer-owned recording and screenshots| artifact["Producer artifact route"]
  decision --> release["Release record"]
```

Desktop recordings are ordinary scenario-to-desktop captures. The capture
store computes the checksum and confines artifact serving to its managed root;
deployment-manager never downloads or stores artifact bytes.

```mermaid
sequenceDiagram
  participant P as Ramp producer
  participant C as Capture store
  participant E as EvidenceService
  participant G as Release gate
  P->>C: run journey and persist recording/screenshots
  C-->>P: artifact IDs, checksums, ordered steps
  P->>E: ReportTargetVerdict(commit, target, refs, disposition)
  E-->>G: reference-backed verdict
  G->>G: match every required target and approval
  G-->>P: allow or refuse with named reason
```
