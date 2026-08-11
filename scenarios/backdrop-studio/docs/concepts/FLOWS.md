# Flows

## 1 · Compose and render a procedural backdrop

The lane that must work with no model, no network, and no `asset-studio`.

```mermaid
sequenceDiagram
    actor Op as Operator
    participant BS as backdrop-studio
    participant BM as brand-manager
    participant IT as image-tools

    Op->>BS: compose(style, brief)
    BS->>BM: resolve $brand.* slots
    alt a slot does not resolve
        BM-->>BS: unknown token
        BS-->>Op: refuse, name the slot (CMP-003)
    else all slots resolve
        BM-->>BS: concrete colours
        BS->>BS: build plan (pure, executes nothing)
        BS-->>Op: plan for review (CMP-001)
        Op->>BS: render(plan, seed, n candidates)
        BS->>BS: draw base scene from seed (SCAF-001)
        BS->>IT: apply treatment chain
        IT-->>BS: treated image reference + execution path
        BS->>BS: measure worst-pixel contrast (LEG-001)
        BS-->>Op: candidates + verdicts
    end
```

The plan is returned **before** anything executes. That is the debuggability
seam: a caller reads exactly what would run without spending anything.

## 2 · Compose and render a guided backdrop

Adds the scaffold, the model, and the disclosure obligation.

```mermaid
sequenceDiagram
    actor Op as Operator
    participant BS as backdrop-studio
    participant IT as image-tools
    participant AG as ai-gateway

    Op->>BS: compose(style[guided], brief)
    BS->>BS: check adapter licence (CMP-006)
    BS->>BS: render scaffold → conditioning image (SCAF-002)
    Note over BS: reserved regions drawn flat (SCAF-003)
    BS->>IT: generate(role, profile, prompt, conditioning)
    IT->>IT: probe host capability
    alt local backend can serve
        IT->>IT: run locally
    else local cannot serve
        IT->>AG: route onward
        AG-->>IT: result
    end
    IT-->>BS: base image + execution path
    BS->>IT: apply treatment chain (RND-001)
    IT-->>BS: treated image reference
    BS->>BS: derive ai_generated = true (REL-001)
    BS-->>Op: candidates + verdicts + execution path
```

Backdrop Studio does not choose between local and routed execution. That decision
belongs to `image-tools`, which already probes the host and holds the routing
policy. This scenario only **records which path ran** (`RND-004`) so a degraded
result is attributable.

## 3 · Judge and release

```mermaid
flowchart TD
    A[Candidate set] --> B{Explicit selection?}
    B -- no --> B1[Refuse: selection names who chose] --> Z[Not released]
    B -- yes --> C{Worst-pixel contrast ≥ threshold?}
    C -- no --> C1[Refuse: state measured, threshold, region] --> S[Offer minimum scrim]
    S --> Z
    C -- yes --> D{Alt text present or marked decorative?}
    D -- no --> Z
    D -- yes --> E{Adapter licence permits commercial use?}
    E -- no --> E1[Refuse: name adapter and restriction] --> Z
    E -- yes --> F{Strategy invoked a model?}
    F -- yes --> G[Release via asset-studio]
    F -- no --> H[Release locally]
    G --> I[Backdrop reference by stable id]
    H --> I
```

Every refusal states its cause and, where one exists, the amendment that would
pass (`UIX-002`). A disabled control without a stated reason is a defect.

## 4 · Consume a released backdrop

```mermaid
sequenceDiagram
    participant LP as landing-page-business-suite
    participant BS as backdrop-studio

    LP->>BS: getBackdrop(id, placement)
    BS-->>LP: uri + surface + reserved_regions + measured contrast + disclosure + alt text
    Note over LP: no image bytes cross this boundary (REL-004)
    LP->>LP: render placement, position copy in overlay region
```

The consumer receives the reserved regions as data, so it positions its own copy
correctly without re-deriving a layout judgement that was already made and
measured.

## 5 · Degradation behaviour

What happens when a dependency is unavailable — stated as behaviour, because a
product that only works with everything running is not deployable.

| Unavailable | Procedural lanes | Model-backed lanes |
|---|---|---|
| `ai-gateway` | unaffected | `image-tools` serves locally if capable; otherwise refuse with cause |
| Local GPU / capable host | unaffected | `image-tools` routes onward if configured; otherwise refuse with cause |
| `asset-studio` | **unaffected** — release locally (`REL-003`) | blocked; refuse with cause |
| `brand-manager` | blocked — no palette lock is possible without the brand | blocked |
| `image-tools` | blocked — every treatment runs there | blocked |

The two hard dependencies are `image-tools` and `brand-manager`. Everything else
degrades to "the procedural catalog still works", which is the posture that makes
this shippable as a desktop product on an ordinary machine.

## Related

- `DOMAINS.md` — which domain owns each step
- `INTEGRATIONS.md` — the seams these flows cross
- `../internal/DECISIONS.md` — why the asset-studio split falls where it does
