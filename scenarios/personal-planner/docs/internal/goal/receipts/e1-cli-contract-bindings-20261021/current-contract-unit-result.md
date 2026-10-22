# Current CLI contract and unit validation result

- Test Genie run `20261021-013929-963ad684`, terminal FAIL, 2 phases: `contracts` PASS; `unit` FAIL.
- `cd packages/proto && make verify-committed-gen` passed before the run.
- `contracts` reports Manifest Contract and Proto Bindings at L3 with no blocking findings for the repaired bindings. The 52 CLI architecture warnings and 9 measure metadata infos remain.
- `unit` failed only in workspace `cli` at `TestPrimitiveEvidenceArtifactCurrent` (`cli/evidence_test.go:31`). Its committed `.vrooli/generated/cli-primitive-evidence.json` no longer matches the repaired manifest; the test's owner-directed remedy is `UPDATE_CLI_EVIDENCE=1`.
- Full `findings.report` artifact: `artifact_018e8df2ed949ecab898e02ab73db26d`, 607,693 bytes, SHA-256 `a53a549494a126ad42c0bafec7543a65e63e26f09e6171227171f92be9990878`.
- Unit phase log artifact: `artifact_839f657b257ca94e8ef071543f25a88e`, 15,560 bytes, SHA-256 `e227fc4550d5d94e7b6ffb870d2467bf01e97521cbe1a6c12f482421b51fd153`.
- The artifacts were read from Test Genie's opaque owner paths. The exact diagnostic is: `committed primitive evidence "../.vrooli/generated/cli-primitive-evidence.json" is stale — regenerate by running this test with UPDATE_CLI_EVIDENCE=1`.
- Do not rerun run `20261021-013929-963ad684`. A bounded follow-up is admitted to regenerate this one owner-generated artifact from the current manifest, then rerun only `contracts,unit` once.

This failed check is retained as evidence. It does not accept the CLI outcome, the full E1 epoch, or E1/Q1.
