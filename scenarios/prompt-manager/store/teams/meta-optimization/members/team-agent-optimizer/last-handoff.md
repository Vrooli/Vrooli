### Domain worked this heartbeat
- team

### Target picked
- `infra-health` - lowest team health at 0.53 with `Role coverage is weak`; agent default skipped because `agent-29` was visited 2026-05-14 and still has pending deprecation decision.

### Disposition
- no-action for infra-health structure; capability-gap for prompt-manager graph/topic validation.

### Evidence
- `graph node infra-health` reports `team-role-coverage=0.00`.
- `team show`, `roles.json`, and `team.json operatingContract.members` all define 3 roles/members with lanes, decisions, writes, safety rules, and task parameters.
- `graph topics --team infra-health` has 0 errors, but warns that `api-core/storage` path prose is a topic leak.

### Expected delta
- Graph health should stop selecting infra-health as weak-role work when the team contract is complete.
- Measure with `prompt-manager graph node infra-health` and `prompt-manager graph topics --team infra-health`.

### Capability architecture
- weak
- Primary layer gap: collection/validation tooling
- Routing: capability-gap/backlog

### Artifacts updated
- TEAM_AUDIT.md: not edited because this run’s write surface only allowed knowledge, decisions, and handoff.
- DEPRECATION_QUEUE.md: unchanged.

### Decisions raised this heartbeat
- `dec-1778884421236613535` - `capability-gap` - Make prompt-manager team graph/topic validation distinguish complete role contracts and storage-path prose from real gaps.

### Knowledge entries written
- `team-visited/infra-health` (`knw-1778884360443005942`)
- `team-audit/2026-05-15` (`knw-1778884374384865969`)
- `friction-inbox/prompt-team-agent-storage/graph-role-coverage-false-positive` (`knw-1778884389368910375`) pending curator routing to `friction-report/prompt-team-agent-storage/...`