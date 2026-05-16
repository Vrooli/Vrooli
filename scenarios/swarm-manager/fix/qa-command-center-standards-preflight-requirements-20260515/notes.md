# QA Evidence

Source: GCT readiness job `0065e5f4-962f-40b7-9bad-8ad6df2240f1` for `command-center`, completed 2026-05-15T22:04:52Z. Overall readiness: yellow. Checks: rules=completed, tests=completed, tidiness=skipped.

Standards dimension: available=true, blockingViolations=4, warnings=17, totalViolations=21. Top critical findings:
- `api/main.go:17`: Missing ScenarioName in preflight config. Recommendation shows adding `preflight.Run(preflight.Config{ScenarioName: "command-center"})` at main start.
- `Makefile`: Scenario Required Structure, recommendation: add required resource at Makefile.
- `requirements/index.json`: P0 target missing requirements, recommendation: link each P0/P1 operational target to at least one requirement before publishing.
- `requirements/index.json`: second P0 target missing requirements with same recommendation.

The tests dimension also failed its standards phase: standards violations exceed fail_on=high, highest=critical.

Success target: rerun `git-control-tower review run command-center --details=10 --json`; standards blockingViolations=0 and tests standards phase no longer fails.