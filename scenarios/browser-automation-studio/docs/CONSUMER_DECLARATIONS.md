# BAS Consumer Declarations

A scenario that adopts Browser Automation Studio keeps its declaration in its own source tree at `.vrooli/browser-automation-studio/consumer-declaration.json`. BAS does not discover, name, or retain consumer-specific configuration.

The file uses `browser-automation-studio.consumer-declaration/v1` and declares stable profile keys, workflow references, allowed variable names, and scalar non-secret preferences. See `schemas/consumer-declaration.schema.json` and `fixtures/consumer-declarations/valid.json`.

Never place credential values, cookies, tokens, proxy passwords, browser storage, or a runtime BAS profile ID in this file. Runtime profile adoption is protected operational state in BAS; the consuming scenario reconciles its public key to that state and owns any one-to-one account policy.

Validate a payload through `browser-automation-studio consumer-declarations validate --declaration-json '<json>'`. The Connect service accepts a payload rather than a filesystem path, so it cannot read arbitrary server files. A consumer may read its own declaration file locally before invoking the command.

To retire an adoption, remove the scenario-owned declaration and explicitly delete or rotate the BAS runtime profile through BAS’s normal operator surface. No relocation, compatibility, or backfill behavior is performed.
