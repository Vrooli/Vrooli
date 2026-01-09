# Research Notes

## Uniqueness Check
- Legacy Test Genie implementation archived at `scenarios/test-genie-old/`; no other active scenario provides AI-driven, coverage-aware test orchestration.
- Related tooling includes `scripts/scenarios/testing/` (shared bash harness) and Vrooli Ascension workflows for integration coverage.

## Related Scenarios & Resources
- `scenario-auditor`, `scenario-completeness-scoring`, and `ecosystem-manager` consume scenario test results and will eventually benefit from a more portable orchestrator.
- Resources required: PostgreSQL (test registry), Ollama (AI prompt generation), optional Redis/Qdrant for performance and semantic search.

## External Inspiration
- Modern test intelligence platforms (Launchable, Testim, QA Wolf) treat coverage + requirement tracking as first-class signals and expose APIs for automation. Use them as references for multi-language orchestration UX.
