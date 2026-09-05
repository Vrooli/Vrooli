# Machine-Readable References

Vrooli documentation uses two machine-readable reference families. They are related, but they solve different problems and should not be collapsed into one syntax.

## Relationship References

Use relationship references when a document intentionally links to another artifact for traceability.

| Syntax | Meaning |
|---|---|
| `[CODE: path/to/file.go]` | This doc is linked to implementation code. |
| `[DOC: docs/reference/api-endpoints.md#health]` | This doc or code is linked to documentation. |
| `[REQ: OT-P0-005]` | This doc is linked to a requirement or operational target. |

Relationship references create auditable edges. Documentation-health tooling can check that the target exists and that code and docs stay connected.

Examples:

```markdown
The docs health runner is implemented in [CODE: scenarios/test-genie/api/internal/docs/runner.go].
See [DOC: docs/reference/cli-commands.md#scenario-management] for the operator command surface.
Covered by [REQ: OT-P0-005].
```

## Marked Inline References

Use marked inline references when an inline literal needs a namespace so scanners do not guess what the string means.

Syntax:

```text
`marker:value`
`marker[qualifier]:value`
```

The marker and qualifier are metadata. They are not part of the literal value. For example, the path value in `path:docs/README.md` is `docs/README.md`, not `path:docs/README.md`.

### Markers

| Marker | Use for |
|---|---|
| `path` | Repo-relative file or directory path. |
| `doc` | Documentation page/path intended for docs navigation. |
| `url` | External URL. |
| `topic` | Prompt-manager knowledge topic prefix or entry topic. |
| `skill` | Prompt-manager skill id. |
| `agent` | Prompt-manager agent id. |
| `team` | Prompt-manager team id. |
| `scenario` | Scenario id. |
| `resource` | Resource id. |
| `action` | Prompt-manager Action id. |
| `decision` | Decision context or decision id. |
| `goal` | Swarm-manager goal name. |
| `cli` | CLI command or subcommand. |
| `env` | Environment variable. |
| `platform` | OS / architecture / runtime target such as `platform:darwin/arm64`. |
| `mime` | MIME or media type such as `mime:application/json`. |
| `route` | HTTP, API, or UI route path. |
| `package` | Package, module, or import path. |
| `literal` | A string that looks machine-readable but should not be semantically validated. |
| `claim` | A documentation enforcement claim that must resolve to a Structure Health catalog claim. |
| `num` | An intentional, owner-backed number in prose that the derived-count lint should not flag. Carries a category (see below): `num[target]:1000`. |

### CLI references

Use `cli:` for Vrooli-owned command references that are meant to describe current executable truth:

```markdown
Run `cli:vrooli scenario test cli-health`.
Check docs with `cli:knowledge-observatory docs health cli-health --checks=refs,commands`.
```

CLI Health is the source of truth for current command validation. Consumers such
as Knowledge Observatory docs health, Prompt Manager action validation, and Plan
Manager validation delegate command existence and argument-shape checks to CLI
Health instead of maintaining their own command catalogs. Knowledge Observatory
also validates conservative Vrooli-owned commands in fenced shell snippets through
its `commands` check.

Qualifiers decide whether a `cli:` reference must exist now:

- Use `cli[future]:...` only for planned commands.
- Use `cli[old]:...` only for historical or deprecated commands.
- Use `cli[external]:...` for non-Vrooli commands.
- Use `cli[literal]:...` when the text is command-shaped but not meant to be validated.
- Use `cli[example]:...` for illustrative examples that are not current operational instructions.

Unqualified `cli:` references are treated as current Vrooli-owned command references.
CLI Health validates them without executing the referenced command. A partial result
means the command path exists, but reliable argument metadata was unavailable.

### Qualifiers

Qualifiers modify validation, not the value.

| Qualifier | Meaning |
|---|---|
| `example` | Illustrative only; tools may syntax-check but should not require current existence. |
| `old` | Historical or deprecated; current existence is not required. |
| `future` | Planned; absence is allowed. |
| `optional` | May or may not exist depending on installation, config, or platform. |
| `external` | Outside this repo or local Vrooli installation. |
| `literal` | Intentionally not semantic despite using a marker-shaped form. |

### Number categories

The `num` marker takes exactly one **category** (written in the qualifier slot) stating *why* the number is owner-backed and therefore exempt from the derived-count lint. These categories are specific to `num`; they are not general-purpose qualifiers.

| Category | Use for |
|---|---|
| `target` | A goal value the system is steering toward (changes by decision, not drift). |
| `threshold` | A limit or gate baked into behavior (rate limit, retry count, timeout). |
| `price` | A monetary amount (SKU price, tier cost). |
| `version` | A pinned version, protocol number, or schema revision. |
| `decision` | A count fixed by an explicit operator decision. |
| `sot` | Mirrors a named source of truth (reserve-and-generate candidate). |

A `num` marker with no recognized category is itself flagged ("intentional number, no stated reason") — tagging must carry a reason, so it stays the documented exception rather than a cheap way to silence the lint.

Examples:

```markdown
Use `topic[example]:friction-inbox/*` for cross-team friction intake.
Use `topic[example]:friction-report/<scope>/*` for routed friction observations.
Generated scenarios may have `path[example]:scenarios/<name>/docs/README.md`.
The target platform is `platform:darwin/arm64`.
The string `literal:if/else` is prose, not a topic or path.
The free tier allows `num[threshold]:100` requests/day; we target `num[target]:1000` paying users.
```

## Choosing The Right Syntax

Use `[CODE:]` when you mean "this document is linked to this implementation."

Use `path:` when you are merely mentioning a path-shaped value.

Use `[DOC:]` when linking documentation as a traceability target.

Use `doc:` when mentioning a documentation path as a typed inline literal.

Use `topic:` when mentioning a current prompt-manager knowledge topic that should resolve to the topic graph. Use `topic[example]:` for illustrative topic strings.

Use `literal:` when the string looks machine-readable but should not be interpreted by scanners.

Use `num[<category>]:` only for a number that is genuinely owner-backed (a target, threshold, price, version, decision, or source-of-truth mirror). The default for a count in prose is to **reword it out** or point at its source of truth — most counts drift. Tagging is the exception, not the reflex. The derived-count lint (`docs health … numbers`) surfaces untagged counts as warnings.

For operational instructions that tell an agent exactly what to run, prefer a full command. For example, use `prompt-manager team knowledge-list meta-optimization --topic-prefix=friction-inbox/` instead of only `topic:friction-inbox/*` when the command itself is the instruction.

## Validation Ownership

This document owns the syntax and vocabulary. Individual tools own domain validation:

- Prompt-manager validates `topic` references against `topics.json`.
- Test-genie validates documentation-phase references such as `path` and `doc` during scenario tests.
- Knowledge-observatory reports documentation audit issues for relationship references and marked inline references.
- Swarm-manager is the source of truth for `goal` references. Automated resolution against `swarm-manager goals list` is target-state; until it exists, owning teams diff their `goal:` references against the live goal list in their own loops (see `docs/agent-system/OPERATING_GRAPHS.md` §"State belongs to scenarios; prose holds judgment").

Scanners should keep inferred heuristic detection as a permanent backstop for unmarked references. A marked reference is the high-confidence form; an unmarked slash-shaped inline string is an inferred hint that tools may validate when it looks relevant.
