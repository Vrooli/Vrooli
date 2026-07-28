# V2 Workflow Migration Status

V2 is the sole supported execution representation: API compilation produces typed proto actions, and the driver dispatches on `ActionType`. Legacy `type`/`params` instruction streams are not a compatibility surface.

Completed: typed ingress, driver parameter extraction, UI consumers, replay timeline evidence, and legacy-stream guard coverage.

Compatibility rule: supported public workflow inputs are normalized before compilation. Tests and mocks must construct typed instructions; code must not reintroduce legacy-shaped fallback fields.

Migration follow-up: keep generated proto outputs fresh and treat package-prefix changes as an intentional versioned contract migration.
