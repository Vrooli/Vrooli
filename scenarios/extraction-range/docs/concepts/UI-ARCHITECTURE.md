# UI Architecture

## Shared contract

Read the [UI Architecture shared guide](/scenarios/template-manager/docs/concepts/UI-ARCHITECTURE.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Use this scenario’s UI manifest and source tree for its actual layout.
Record local layout or adoption constraints here; the current template guide
does not establish that an older scenario has migrated.

This scenario does not declare app-delivery capability: it has no intentional browser push destination or app-delivery producer. The shared initializer and settings component are therefore omitted. Scenario-specific notification behavior, if any, remains owned by its own feature contract.
