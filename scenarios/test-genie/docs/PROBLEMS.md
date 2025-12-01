# Open Issues
- Legacy documentation claims high coverage and complete P0 features even though v2 rewrite currently contains only template code.
- The scenario still needs to rebuild the CLI delegation workflow that triggers suite generation remotely; existing Go backend lives in `scenarios/test-genie-old/` for reference.
- Requirement modules have their first unit-level validation wired to `api/suite_requests_test.go`, but integration + CLI acceptance coverage remain gaps until OT-P0-002 fully lands.

# Deferred Ideas
- Evaluate which pieces of the archived Go services or CLI should be ported verbatim versus redesigned.
- Consider generating migration scripts to import existing test suites from the legacy implementation for benchmarking.
