import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { HealthPage } from "./index";
import { selectors } from "../../consts/selectors";
import * as api from "../../lib/api";
import type { SelfHealth } from "../../lib/api";

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <HealthPage />
    </QueryClientProvider>
  );
}

describe("HealthPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders catalog, conformance and ledger from a populated snapshot", async () => {
    const snapshot: SelfHealth = {
      catalog: { totalPhases: 18, delegatedPhases: 13, nativePhases: 5, phases: [{ name: "smoke", delegated: false }] },
      conformance: [
        { provider: "proto-health", phase: "proto", reachable: true, contractValid: true, specValid: true, identityOk: true, metricsAdopted: true, adoptionScore: 1 }
      ],
      conformanceFreshness: "live",
      ledger: {
        windowDays: 30,
        runCount: 42,
        availability: 0.9,
        runOutcomes: [{ outcome: "passed", count: 40 }],
        phases: [{ phase: "performance", availability: 1, failureRate: 0, totalObservations: 10, metricsAdopted: 10, duration: { p50: 19, p95: 83, max: 383 } }]
      }
    };
    vi.spyOn(api, "getSelfHealth").mockResolvedValue(snapshot);

    renderPage();

    await waitFor(() => expect(screen.getByTestId(selectors.health.catalog)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.health.conformance)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.health.ledger)).toBeInTheDocument();
    expect(screen.getByText("proto-health")).toBeInTheDocument();
    expect(screen.getByText("performance")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.health.empty)).not.toBeInTheDocument();
  });

  it("shows the empty-ledger fallback when no runs are recorded", async () => {
    vi.spyOn(api, "getSelfHealth").mockResolvedValue({
      catalog: { totalPhases: 0, phases: [] },
      ledger: { runCount: 0, phases: [] }
    });

    renderPage();

    await waitFor(() => expect(screen.getByTestId(selectors.health.empty)).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.health.ledger)).not.toBeInTheDocument();
  });

  it("surfaces an error state when the fetch fails", async () => {
    vi.spyOn(api, "getSelfHealth").mockRejectedValue(new Error("boom"));

    renderPage();

    await waitFor(() => expect(screen.getByText(/Failed to load self-health/)).toBeInTheDocument());
  });
});
