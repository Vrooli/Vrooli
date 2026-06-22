import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  makeAdviseEnginesResponse,
  makeAnalyzeMigrationsResponse,
  makeEngineCandidate,
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

describe("AdvisorView accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    adviseEngines.mockResolvedValue(
      makeAdviseEnginesResponse({
        scenarioCount: 1,
        candidates: [makeEngineCandidate({ blockers: ["needs postgres"] })],
      }),
    );
    analyzeMigrations.mockResolvedValue(makeAnalyzeMigrationsResponse());
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the engine-fitness tab without axe violations", async () => {
    const { container } = renderWithProviders(<AdvisorView />, { routerEntries: ["/advisor"] });
    await waitFor(() =>
      expect(screen.getByTestId(selectors.advisor.candidate({ scenario: "demo" }))).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
