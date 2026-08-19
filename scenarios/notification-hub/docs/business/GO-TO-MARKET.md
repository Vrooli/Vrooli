# Go To Market — Notification Hub

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

**Audience.** Two groups, in this order.

1. **Self-hosters who already run a push pipe.** People with ntfy, Gotify,
   or Pushover wired into scripts and home infrastructure. They have
   solved transport and have not solved routing: everything fires
   immediately, at any hour, with no deduplication and no way to answer
   back. They are technical, vocal, and already convinced that running
   things themselves is worth effort.
2. **People running AI agents that work unattended.** Anyone whose agent
   takes minutes or hours and occasionally needs a decision. They are
   currently gluing Slack webhooks to approval callbacks and are aware
   that it is glue.

**Positioning.** *The routing brain for your own machines and devices.*
Not a notification API, and not another push pipe — the layer that
decides whether to interrupt you, on which device, and relays through
another of your machines when this one cannot reach you.

**Main claim.** Your agents can reach you, on the right device, without
paying per message and without your notifications leaving your hardware.

**Proof needed.** One recorded end-to-end run: an agent pauses, the owner
receives a push on a locked iPhone, answers from the notification, and
the agent resumes with that answer. That single artifact carries the
positioning better than any amount of copy, and it is exactly what the
acknowledgement work in `MONETIZATION.md` unlocks.

## Channels

Mapped to the project channel registry. Ranked by fit, not by reach.

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| `oss-discovery` | **Strongest fit.** The ntfy/Gotify/Apprise audience is our audience, and the honest wedge is complementary rather than competitive: "great pipe, here is the brain on top of it." Self-hosted communities carry a local-first story on their own. | A comparison page that credits the pipes rather than attacking them; a working config that sits on top of ntfy in under ten minutes. | Inbound installs traceable to self-hosted communities; unsolicited comparisons written by other people. |
| `skill-registries` | Publish a *notify the human* / *ask the human* skill so agents in Claude Code, Codex, and Cursor can reach a person through the hub. Near-zero marginal cost, and it pulls Vrooli installs behind it. | A published skill in the `skills/` publication source; a one-command local setup path. | Skill installs, and the ratio of skill users who go on to install Vrooli. |
| `in-product-expansion` | Every Vrooli scenario that finishes long work is a placement. The hub becomes discoverable because something the owner already runs starts telling them things. | A one-call integration pattern other scenarios can copy; a default surfaced on first long-running completion. | Count of other scenarios routing through the hub without being asked to. |
| `community-content` | The agent-oversight problem is topical and under-served. The 4,000-email incident is the kind of concrete failure that carries a technical argument. | A written teardown of where approval gates belong for autonomous agents, using this scenario as the worked example. | Referral traffic and inbound questions about the approval gate specifically. |
| `web-seo` | Long-tail intent exists — self-hosted push comparisons, agent approval patterns — but it converts slowly and rewards content we have not written. | Comparison and how-to pages. | Organic impressions on self-hosted-alerting queries. |
| `app-stores` | Not applicable. Delivery rides on providers' existing apps; shipping our own would mean an Apple Developer account, a Mac signing path, and an APNs certificate for no user-visible gain. | None. | None. |

## Launch Motion

1. Deliver one real notification to the owner's iPhone (OT-P0-001). No
   positioning work happens before this, because everything the previous
   scenario ever did was simulated and the claim has to be earned once.
2. Build the routing core — devices, preferences, quiet hours, duplicate
   suppression, retry. This is the part that differentiates from a pipe.
3. Ship acknowledgement, the `ask` primitive, and escalation as one slice
   (OT-P1-009 through OT-P1-011). This is the part that differentiates
   from everything else, and it is the gate on any positioning that
   mentions agents. It needs no second machine, which is why it precedes
   the fleet work rather than following it.
4. Record the proof artifact: agent pauses, human answers from a locked
   phone, agent resumes.
5. Open `oss-discovery` and `skill-registries` together — the skill gives
   the community something to run, the comparison page tells them why.
6. Decide the add-on question only after acknowledgement has served a
   real approval gate for a real run.

Cross-node relay through `vrooli-bridge` is a strong demo but not a
launch gate: it needs a second machine, which most of the audience will
not have on day one.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Your agents can reach you — and can wait for your answer." | Agent operators | The proof artifact from step 4. | blocked on OT-P1-009 through OT-P1-011 |
| "A routing brain for the push pipe you already run." | Self-hosters | Working ntfy-backed config; quiet hours and dedupe demonstrable. | ready after step 2 |
| "It never leaves your hardware, and it never bills per message." | Both | Zero declared resource dependencies; no metering anywhere in the design. | **ready now** — this is true today and structurally hard for metered competitors to match |
| "Reaches the device even when this machine cannot." | Fleet owners | A relayed delivery through a second node. | blocked on OT-P1-001 |
| "It knows when not to tell you." | Both | Quiet hours, dedupe, digest collapsing. | ready after step 2 |

Avoid claiming multi-tenant, team, or workspace capability. It is not
built, it is explicitly a non-goal, and inviting that audience produces
requests that pull the scenario back toward the rejected charter.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Publish the *ask the human* skill and measure installs | `skill-registries` | To be set with the monetization team. | If installs materially exceed Vrooli installs, the agent-oversight framing leads and the bundle framing follows. |
| Comparison page positioned as complementary to ntfy | `oss-discovery` | To be set with the monetization team. | If self-hosters adopt but never use acknowledgement, the product is a better pipe, not an oversight layer — and the add-on hypothesis dies. |
| Track acknowledgement rate against delivery rate | in-product telemetry | Acknowledgement rate must not decay over time. | A falling rate means the spine is being muted. Fix fatigue before any pricing conversation. |
| Count other scenarios integrating without being asked | `in-product-expansion` | To be set with the monetization team. | Organic internal adoption is the cleanest evidence that the capability is load-bearing rather than nice to have. |

Thresholds are deliberately unset. Inventing them here would create
numbers with no provenance; they belong to the monetization team's
taxonomy.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — market position, packaging, and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes and sequencing
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
- Channel registry: `path:../../docs/monetization/catalogs/channels/README.md`
- Publication-source skills: `path:skills/README.md`
