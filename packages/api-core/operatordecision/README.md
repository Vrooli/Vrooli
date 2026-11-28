# operatordecision

Domain-neutral substrate for asking the operator a question and acting on the answer. A scenario
raises `<source>.<kind>.decision_requested.v1` (`Facts`, `Raiser`); Notification Hub turns it into an
ask; the answer returns as `notification_hub.ask.resolved.v1`, which the scenario's `Webhook` hands to
its `DomainEffect`s. See `decision.go` for the contract and the other files for raising, withdrawing,
context requests and the pending store.

A decision is a page plus a packet. `Facts.Explainer` carries the page (`NewPage`; `CoverPage` is the
floor page the Hub shows when a producer sends none). `Facts.Packet` is the absolute path of the brief
that carries out the answer (`PacketPathProblem` states the rule); it returns in the resolution.
`Facts.Questions` is an answer form under the options (choice and text questions). The page's
`Groups(questionID, ...)` switches fill its `multi_choice` question, and the answers return as
`DecisionResolution.Responses`, read with `Response(id)`. They come from the Hub's record once
the resolution is verified. Further page parts: `Proportion`, `Command`, `Recurrence`, `Matrix`
(`page_parts.go`).

## Resolutions are confirmed with the Hub

The webhook body is a claim, not evidence. vrooli-events signs what it delivers, but it accepts an
event from any caller with any `source`, so anything able to reach the bus could publish a
`notification_hub.ask.resolved.v1` naming a real pending ask. `NewWebhook` therefore reads the ask
back from Notification Hub, as the scenario's own ask-reader principal, before any effect runs:

```go
operatordecision.NewWebhook(operatordecision.WebhookConfig{
    Secret:  operatordecision.StaticSecret(secret),
    Source:  "storage-manager",                                  // the Hub's Ask.Source for this scenario
    Reader:  askreader.NewReader("storage-manager", credential), // credential: the ask-reader credential
    Effects: []operatordecision.DomainEffect{effect},
})
```

For a resolution the Hub must say the ask is the scenario's own, closed, and that the state, answer,
answerer, producer event, requester reference, subject key and correlation are what the body claims
(`VerifiedResolution`). The effect receives the **Hub's** record, so the correlation that locates the
scenario's own pending record (`authorization_request_id`, `run_id`, `approval_id`, ...) is the one the
scenario raised, never one a forger chose. A context request is confirmed the same way
(`VerifiedContextRequest`), except that its correlation and requester reference are taken from the Hub without being compared: an open ask the producer revised since the event legitimately carries newer ones.

| Outcome | Response | What happens |
|---|---|---|
| Hub confirms | effect runs | counted `Verified` |
| Ask is another scenario's (body `source`) | 200 `ignored` | no read; every scenario sees every resolution |
| Hub holds no such ask, still has it open, or disagrees with the body | 409 | refused, logged, counted by reason; the bus redelivers and dead-letters it where an operator can retry it |
| Hub or authenticator cannot be read | 503 | counted `Unavailable`; the bus redelivers (5 attempts, 1 s to 16 s apart, then dead letter), so a real approval is delayed, never lost; `POST /api/v1/subscriptions/{id}/deliveries/retry` requeues a dead letter |
| Scenario has no ask-reader credential yet | effect runs on the body | counted `UnverifiedApplied` and logged on every resolution with the operator's command |

The last row exists so the rollout cannot break live approvals: it ends the moment the operator runs
`scenario-authenticator service-principals sync --apply`, which issues each scenario's
`<source>-ask-reader` credential (declared in its `.vrooli/service-principals.json`). `Webhook.Stats()`
reports the counts. A scenario with no `Reader` is on the unverified path by construction.

### Telling the operator

A 409 on a resolution is a security event: something tried to apply an answer the operator did not give.
Left to the bus it would sit in a dead-letter list. `WebhookConfig.OnRefused` is called, once per delivery,
for each resolution the Hub's own record contradicts (`mismatch`, or `ask_open`: the body names a closed
ask the Hub still has open). It is not called for an unreadable Hub (503), an ask the Hub does not hold
for the scenario, a context request, or a resolution applied without a credential.

`RefusalNotifier.Notify` is the implementation every consumer passes
(`hubidentity.WebhookConfig`, which also gives the webhook its ask reader; a guard test fails any scenario that builds `operatordecision.NewWebhook` without it). It sends one notice through the scenario's own
`<source>-notification-hub` sender principal (`hubsender.Client`), urgency high, sensitivity private:

> Something tried to apply an answer you didn't give for "<the ask's question>". It was refused; nothing
> changed.

with the ask's page (`/asks/<id>`) as the notice's tappable context link and, under "Technical details", which scenario refused which ask and why.
The question is the Hub's record of the ask (`askreader.Ask.Question`, truncated to one line), never a
field of the event body, and the text names no answer, answerer or other claim the body made.

- **Once per refusal.** The idempotency key is `refused-resolution:<source>:<ask>:<reason>:<question hash>`,
  so the six redeliveries of one refusal send one notice, across restarts too. A refusal whose notice could
  not be sent is not remembered, so the next redelivery sends it. An ask the producer revised is a new
  refusal, because the Hub refuses to reuse a key with different terms.
- **Bounded.** After five notices per scenario in ten minutes (`DefaultNoticeBurst`,
  `DefaultNoticeWindow`) one summary notice, "Many answers you didn't give were refused", replaces the
  rest, and nothing more is sent until the window passes. `RefusalNotifier.Stats()` counts sent notices,
  summaries, suppressed refusals and failed sends.
- **Needs the sender credential.** Without `<source>-notification-hub` issued the refusal is still
  refused, logged and counted; the log names the operator's command. It is issued by the same
  `service-principals sync --apply` and found on the next refusal, with no restart.

Operator order, with one restart of each service and nothing after the apply:

1. Restart Notification Hub on the version that returns `Ask.correlation` (a Hub that predates the field
   returns no correlation, so a confirmed resolution would be refused as a mismatch), then vrooli-events,
   then the consumers and publishers, so each runs the new code.
2. Run `scenario-authenticator service-principals sync --apply`.

Nothing restarts after the apply. A consumer reads its credential when it reads an ask, and a missing one
is not remembered, so its next resolution after the apply is read back from the Hub (`Verified`, no longer
`UnverifiedApplied`). Event publishers look again for a missing credential once a minute, so each starts
presenting its identity within a minute (`eventbus.PublisherAuthorization`). Apply only after the Hub has
restarted: a consumer that is verifying against an old Hub refuses genuine resolutions.

Tests: `webhook_verify_test.go` (forged body of a pending ask, forged fields, wrong source, Hub outage
then redelivery, no credential), `webhook_lazy_credential_test.go` (a credential issued while the scenario
runs turns verification on with no restart), `refusal_notice_test.go` (one notice across redeliveries, the burst
cap, the Hub's own text, no notice for an outage) and, against the real Hub, notification-hub
`handlers/conversations/resolution_readback_test.go`.
