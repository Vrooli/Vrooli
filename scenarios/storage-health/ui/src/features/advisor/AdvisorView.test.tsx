import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import {
  makeAdviseEnginesResponse,
  makeAnalyzeMigrationsResponse,
  makeEngineCandidate,
  makeMigrationHygiene,
} from "../storage/mocks/factories";

const { adviseEngines, analyzeMigrations } = vi.hoisted(() => ({
  adviseEngines: vi.fn(),
  analyzeMigrations: vi.fn(),
}));

vi.mock("../../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/storage")>();
  return {
    ...actual,
    storageClient: { ...actual.storageClient, adviseEngines, analyzeMigrations },
  };
});

import { AdvisorView } from "./AdvisorView";

beforeEach(() => {
  adviseEngines.mockResolvedValue(
    makeAdviseEnginesResponse({
      scenarioCount: 2,
      candidates: [
        makeEngineCandidate({ scenario: "low", fitnessScore: 0.3 }),
        makeEngineCandidate({
          scenario: "high",
          fitnessScore: 0.9,
          blockers: ["uses postgres extensions"],
        }),
      ],
    }),
  );
  analyzeMigrations.mockResolvedValue(
    makeAnalyzeMigrationsResponse({
      scenarioCount: 5,
      withMigrationsCount: 2,
      debtCount: 1,
      entries: [makeMigrationHygiene({ scenario: "debtor", migrationDebt: 3 })],
    }),
  );
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AdvisorView", () => {
  it("renders engine candidates sorted strongest-first", async () => {
    renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.advisor.candidate({ scenario: "high" }))).toBeInTheDocument();
    });
    const cards = screen.getAllByTestId(/^advisor-candidate-/);
    expect(cards[0]).toHaveAttribute("data-testid", "advisor-candidate-high");
  });

  it("shows blockers on a candidate when present", async () => {
    renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await waitFor(() => {
      expect(screen.getByText(/uses postgres extensions/)).toBeInTheDocument();
    });
  });

  it("switches to the migration hygiene tab and lists debt", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await user.click(screen.getByTestId(selectors.advisor.tab({ tab: "migrations" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.advisor.migrationsSummary)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.advisor.migration({ scenario: "debtor" }))).toBeInTheDocument();
  });

  it("renders the engine empty state when there are no candidates", async () => {
    adviseEngines.mockResolvedValue(makeAdviseEnginesResponse({ scenarioCount: 0 }));
    renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.advisor.enginesEmpty)).toBeInTheDocument();
    });
  });

  it("renders the migrations empty state when there is no debt", async () => {
    const user = userEvent.setup();
    analyzeMigrations.mockResolvedValue(
      makeAnalyzeMigrationsResponse({ scenarioCount: 3, withMigrationsCount: 1, debtCount: 0 }),
    );
    renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await user.click(screen.getByTestId(selectors.advisor.tab({ tab: "migrations" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.advisor.migrationsEmpty)).toBeInTheDocument();
    });
  });

  it("renders the engine error state on failure", async () => {
    adviseEngines.mockRejectedValue(new Error("boom"));
    renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.advisor.error)).toBeInTheDocument();
    });
  });
});
