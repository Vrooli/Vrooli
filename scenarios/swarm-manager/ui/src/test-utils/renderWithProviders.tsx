// Canonical provider entry point for Swarm Manager UI tests. Keep the
// implementation in api-base so provider behavior stays consistent across
// scenarios while the testing contract has a stable local path to validate.
export { renderWithProviders, renderWithProviders as render } from "@vrooli/api-base/testing";
