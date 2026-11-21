# Skill Set

## North Star

Every owed skill role is declared, dialect-correct, reachable, and connected to the sensor or program it claims to improve.

## The rungs and their gates

L0 means service metadata cannot be read. L1 means role declarations are inspectable. L2 means declared sources are reachable and the role assessment is clean.

## What each finding means

`skill-set.service_missing` and `skill-set.service_malformed` identify unreadable scenario metadata. `skill-set.source_missing` identifies a declared skill source that is not present.

## The canonical fix

Repair `.vrooli/service.json`, restore the declared source file, or record a dated waiver with a reason when the role is intentionally not owed.

## How to verify

Run `prompt-manager skill-set validate <scenario> --json`, then run `test-genie provider-contract check skill-set prompt-manager --json`.
