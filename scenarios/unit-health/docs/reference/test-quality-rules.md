# Test-quality enforcement reference

Generated from testquality/catalog.json (catalog 1). Do not edit by hand.

Declared profiles and calibration cases are not certification. Checked clean applies only to the named supported check; unknown is never a pass. Promotion requires separate recorded evidence.

<a id="rule-assertion-observation"></a>

## assertion-observation

Recognize supported assertion observations without claiming behavioral adequacy.

- Version: 1
- Implementation: implemented
- Profiles: go-syntax-v1, react-vitest-v2
- Test kinds: unit, local-integration
- Required evidence: profile-specific-source-or-native-runtime-observation
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: true
- False-positive budget: 0.0500
- Promotion decisions: 1
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C001, C002, C003, C004, C005, C006, C007, C008, C009, C010, C011, C012, C018, C019, C020, C021, C022, C023, C024, C027, C028, C029, C030, C031, C032, C037, C040

A violation is scoped to the declared rule and profile; missing or unsupported evidence is unknown. Native runtime support is implemented for Vitest 2.1.9 only; the Go source profile observes import-bound testing/testify failure mechanisms with package/build-scoped local helper traversal (depth `num[threshold]:8`, `num[threshold]:4096` nodes per body, `num[threshold]:1000` files, `num[threshold]:1` MiB per file and `num[threshold]:16` MiB total). Literal subtests have separate identities. Unresolved calls, cycles and parse failures remain unknown; TestMain, build-excluded tests and supported registered subprocess helpers are not applicable. Static observations are neither executed assertion counts nor negative-path coverage. The public reporter observes requireAssertions with direct, delegated axe, parameterized, local-expect concurrent, custom-matcher, setup, extended and returned-async fixtures. Bare expect, tautologies and setup assertions can satisfy the check without proving matcher completion or behavioral adequacy. Node assert and other alternate assertion libraries are not observed by this native check. Skips and expected-failure inversion remain unknown; failed setup and other failures without assertion-absence evidence remain unknown. Retries retain final state, retry count and prior errors, not fabricated per-attempt history. Command-unique run identities reject stale artifacts. Other installed versions, including api-base's Vitest 1.x profile, are unsupported until independently probed.

<a id="rule-focused-test"></a>

## focused-test

Detect an executable focused-test declaration, not words in strings.

- Version: 1
- Implementation: implemented
- Profiles: vitest-syntax-1.6.9
- Test kinds: unit, local-integration
- Required evidence: source-bound-quality-health-native-lint
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: true
- False-positive budget: 0.0000
- Promotion decisions: 0
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C035, C036, C037

A violation is scoped to the declared rule and profile; missing or unsupported evidence is unknown.

<a id="rule-malformed-expectation"></a>

## malformed-expectation

Detect a supported expectation without a completed matcher.

- Version: 1
- Implementation: implemented
- Profiles: vitest-syntax-1.6.9
- Test kinds: unit, local-integration
- Required evidence: source-bound-quality-health-native-lint
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: true
- False-positive budget: 0.0000
- Promotion decisions: 0
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C021, C025, C028, C031

A violation is scoped to the declared rule and profile; missing or unsupported evidence is unknown.

<a id="rule-async-assertion"></a>

## async-assertion

Require supported asynchronous matchers to be awaited or returned.

- Version: 1
- Implementation: implemented
- Profiles: vitest-syntax-1.6.9
- Test kinds: unit, local-integration
- Required evidence: source-bound-quality-health-native-lint
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: true
- False-positive budget: 0.0000
- Promotion decisions: 0
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C038, C039

A violation is scoped to the declared rule and profile; missing or unsupported evidence is unknown.

<a id="rule-skip-declaration"></a>

## skip-declaration

Report static skip declarations separately from runtime skipped outcomes.

- Version: 1
- Implementation: implemented
- Profiles: go-syntax-v1, vitest-syntax-1.6.9
- Test kinds: unit, local-integration
- Required evidence: source
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: true
- False-positive budget: 0.0000
- Promotion decisions: 0
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C013, C014, C034

Go source declarations and Vitest test.skip declarations are not runtime skipped outcomes. A sole direct import-bound testing.T Skip, Skipf or SkipNow statement, or a skip-only Vitest body, is an unconditional placeholder violation. Conditional and other direct declarations remain unknown/not-executed. Nested callbacks own their declarations separately; helper traversal and dynamic skips are not covered, and absence of a declaration row is not a clean skip assessment. Build-excluded tests and registered process helpers remain not applicable. The scan is bounded to `num[threshold]:4096` nodes per body. C013/C014 and the Vitest C034 fixture calibrate the supported source profiles. No blocking promotion is implied.

<a id="rule-requirement-link"></a>

## requirement-link

Reconcile actual registry IDs and declared validation ownership.

- Version: 1
- Implementation: implemented
- Profiles: go-syntax-v1, react-vitest-v2
- Test kinds: unit, local-integration
- Required evidence: source, requirement-registry
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: true
- False-positive budget: 0.0000
- Promotion decisions: 0
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C057, C058, C059, C060, C061, C062

The typed traceability report separates exact registry membership, declared validation responsibility and per-test execution; it does not derive requirement completion. Current Test Genie declarations are authoritative; unavailable, partial and old-schema registries remain unknown. Go source comments and literal subtest names declare links only. Supported fresh Go JSON execution and Vitest 2.1.9 observations retain pass/fail/skip separately. Stale runs, cached assessments, truncated captures and ambiguous Go name mappings cannot establish current passes. Integration/business-only responsibilities do not imply unit-tag obligations; unspecified responsibility is unknown. Cross-language tag grammar is checked with shared conformance fixtures. Missing runtime evidence is not a clean assessment.

<a id="rule-reliability-cohort"></a>

## reliability-cohort

Compare outcomes only within compatible input cohorts; variation is suspected instability.

- Version: 1
- Implementation: planned
- Profiles: go-runtime-v1, react-vitest-v2
- Test kinds: unit, local-integration
- Required evidence: runtime-outcomes, evidence-identity
- Severity: warning
- Default enforcement: advisory
- Eligible for promotion: false
- False-positive budget: 0.0000
- Promotion decisions: 0
- Promotion prerequisites: reviewed-calibration, reviewed-holdout, native-profile-conformance, false-positive-budget, owner-promotion-decision
- Calibration cases: C063, C064, C065, C066, C067

A violation is scoped to the declared rule and profile; missing or unsupported evidence is unknown.
