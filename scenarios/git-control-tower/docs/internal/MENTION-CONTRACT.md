# Mention-driven advisory contract

Switchboard or the host-ingress owner must authenticate a delivery before GCT
sees it. The normalized envelope contains provider, event, optional event-version, actor, thread,
repository, and head-revision identities plus `authenticated`,
`actorCanRead`, `replyPermitted`, and `botActor` standing.

GCT accepts only addressed `@vrooli summarize`, `@vrooli review`, and
`@vrooli explain` commands. Comment text is untrusted data. Unknown commands,
bot events, forged deliveries, incomplete revisions, and duplicate event keys
are refused before inference or publication. A request is bound to the exact
head revision and repository visibility supplied by the owner.

An edited host comment must carry a new `eventVersion`. Its normalized key is
`provider:eventId:eventVersion`, so an edited command is a new auditable
request rather than an accidental duplicate of the original delivery.

Reply owners should use the revision-bound outbox claim. A claim is refused
when the current host head differs from the request's reviewed head, preserving
the request for audit without publishing a stale answer.

Reply authority is independent from event authenticity. Public rendering uses
only public-safe evidence and redacts private work references. A durable
outbox/owner reply operation must reconcile delivery after interruption; GCT
does not implement provider webhook transport here.
