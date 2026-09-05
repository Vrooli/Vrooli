import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { FrictionOverviewCard } from "../../src/features/stats/components/operational/FrictionOverviewCard.js";
import { renderWithProviders } from "@vrooli/api-base/testing";

const mocks = vi.hoisted(() => ({
  external: vi.fn(), rereads: vi.fn(), findings: vi.fn(), helpRecovery: vi.fn(), repeatedWork: vi.fn(), retry: vi.fn(), definitions: vi.fn(),
}));
vi.mock("../../src/features/stats/api/statsClient.js", () => ({
  fetchExternalToolShare: mocks.external, fetchFileRereadRate: mocks.rereads, fetchFindingRecurrenceRate: mocks.findings,
  fetchHelpRecoveryRate: mocks.helpRecovery, fetchRepeatedWorkRate: mocks.repeatedWork, fetchRetryRate: mocks.retry,
}));
vi.mock("../../src/features/stats/hooks/useMeasureDefinitions.js", () => ({ useMeasureDefinitions: mocks.definitions }));
vi.mock("../../src/features/stats/hooks/useTimeWindow.js", () => ({ useTimeWindow: () => ({ preset: "24h" }) }));

const valid = { validity: { state: "available" as const, reason: "fixture", sampleSize: 20, largestFingerprintShare: 0.25 }, definitionId: "friction.fixture", executedQuery: "SELECT" };

afterEach(() => vi.clearAllMocks());

test("friction overview renders all durable measures and retry evidence", async () => {
  mocks.definitions.mockReturnValue({ data: [] });
  mocks.external.mockResolvedValue({ ...valid, share: 0.1, externalCalls: 2, resolvedCalls: 1, unknownCalls: 1 });
  mocks.retry.mockResolvedValue({ ...valid, rate: 0.2 });
  mocks.helpRecovery.mockResolvedValue({ ...valid, rate: 0.3 });
  mocks.repeatedWork.mockResolvedValue({ ...valid, rate: 0.4 });
  mocks.rereads.mockResolvedValue({ ...valid, rate: 0.5, filesReadMoreThanOnce: 2, readCalls: 4 });
  mocks.findings.mockResolvedValue({ ...valid, rate: 0.6, recurringFindings: 2, totalFindings: 3, recurringFingerprints: 1 });

  renderWithProviders(createElement(FrictionOverviewCard));
  await waitFor(() => assert.equal(screen.getByTestId("friction-overview-values").textContent?.includes("10.0%"), true));
  assert.equal(screen.getByTestId("friction-overview-card").textContent?.includes("Largest retry fingerprint: 25.0%"), true);
  assert.equal(screen.getByTestId("friction-overview-card").textContent?.includes("1 resolved calls"), true);
});

test("friction overview exposes query failures", async () => {
  mocks.definitions.mockReturnValue({ data: [] });
  mocks.external.mockRejectedValue(new Error("friction unavailable"));
  mocks.retry.mockResolvedValue({ ...valid, rate: 0 });
  mocks.helpRecovery.mockResolvedValue({ ...valid, rate: 0 });
  mocks.repeatedWork.mockResolvedValue({ ...valid, rate: 0 });
  mocks.rereads.mockResolvedValue({ ...valid, rate: 0, filesReadMoreThanOnce: 0, readCalls: 0 });
  mocks.findings.mockResolvedValue({ ...valid, rate: 0, recurringFindings: 0, totalFindings: 0, recurringFingerprints: 0 });
  renderWithProviders(createElement(FrictionOverviewCard));
  await waitFor(() => assert.equal(screen.getByRole("alert").textContent, "Failed to load friction metrics: friction unavailable"));
});
