import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { EvidencePage, ScenarioExplorerPage } from "./ExperiencePages";

const mocks = vi.hoisted(() => ({
  fetchEvidence: vi.fn(),
  fetchScenarioSpec: vi.fn(),
  recaptureScenario: vi.fn(),
}));

vi.mock("../api/experience", async () => {
  const actual = await vi.importActual<typeof import("../api/experience")>("../api/experience");
  return {
    ...actual,
    fetchEvidence: mocks.fetchEvidence,
    fetchFleet: vi.fn(),
    fetchScenarioSpec: mocks.fetchScenarioSpec,
    recaptureScenario: mocks.recaptureScenario,
  };
});

describe("ScenarioExplorerPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders live page and machine-claim data", async () => {
    mocks.fetchScenarioSpec.mockResolvedValueOnce([
      {
        document: { id: "fleet", title: "Fleet", path: "pages/fleet.json", status: "active" },
        spec: {
          page: { id: "fleet", title: "Fleet" },
          priorities: [{ statement: "Coverage first." }],
          states: [
            { id: "default", description: "Default." },
            { id: "empty", description: "Empty." },
          ],
          elements: [{ id: "debt-table", role: "table", name: "Experience debt" }],
          claims: [
            {
              id: "debt-table-perceivable",
              type: "element-present",
              tier: "machine",
              elements: ["debt-table"],
              states: ["default"],
            },
            {
              id: "operator-confidence",
              type: "custom",
              tier: "manual",
            },
          ],
        },
      },
    ]);

    renderWithProviders(<ScenarioExplorerPage />);

    const grid = screen.getByTestId(selectors.experience.explorer.depthGrid);
    await waitFor(() => expect(grid).toHaveTextContent("Fleet"));
    expect(grid).toHaveTextContent("L3");
    expect(grid).toHaveTextContent("-");
    expect(screen.getByTestId(selectors.experience.explorer.claimList)).toHaveTextContent("debt-table-perceivable");
    expect(screen.getByTestId(selectors.experience.explorer.claimList)).not.toHaveTextContent("operator-confidence");
    expect(screen.getByTestId(selectors.experience.explorer.evidenceLink)).toHaveAttribute(
      "href",
      "/scenarios/experience-manager/pages/fleet/evidence",
    );
  });

  it("renders loading state while the spec request is pending", () => {
    mocks.fetchScenarioSpec.mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(<ScenarioExplorerPage />);

    expect(screen.getByTestId(selectors.experience.explorer.depthGrid)).toHaveTextContent(
      strings.experience.explorer.loadingSpec,
    );
    expect(screen.getByTestId(selectors.experience.explorer.claimList)).toHaveTextContent(
      strings.experience.explorer.loadingClaims,
    );
  });

  it("renders empty state when the scenario has no pages", async () => {
    mocks.fetchScenarioSpec.mockResolvedValueOnce([]);

    renderWithProviders(<ScenarioExplorerPage />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.explorer.depthGrid)).toHaveTextContent(
        strings.experience.explorer.emptySpec,
      ),
    );
    expect(screen.getByTestId(selectors.experience.explorer.claimList)).toHaveTextContent(
      strings.experience.explorer.emptyClaims,
    );
  });

  it("renders error state when the spec request fails", async () => {
    mocks.fetchScenarioSpec.mockRejectedValueOnce(new Error("server unavailable"));

    renderWithProviders(<ScenarioExplorerPage />);

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(strings.experience.explorer.loadError),
    );
  });
});

describe("EvidencePage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders persisted evidence rows and focuses their accessibility payloads", async () => {
    mocks.fetchEvidence.mockResolvedValueOnce([
      {
        id: "row-1",
        scenario: "experience-manager",
        page: "fleet",
        route: "/",
        state: "default",
        claim: "summary-before-table",
        claimType: "element-present",
        verdict: "passed",
        captureRef: "/artifacts/capture.png",
        axNodeJson: JSON.stringify({ role: "heading", name: "Experience depth" }),
        message: "summary found",
        checkedAt: new Date().toISOString(),
      },
      {
        id: "row-2",
        scenario: "experience-manager",
        page: "fleet",
        route: "/",
        state: "default",
        claim: "debt-table-perceivable",
        claimType: "element-present",
        verdict: "failed",
        captureRef: "/artifacts/capture.png",
        axNodeJson: JSON.stringify({ role: "table", name: "Experience debt" }),
        message: "table missing",
        checkedAt: new Date().toISOString(),
      },
    ]);

    renderWithProviders(<EvidencePage />, {
      routerEntries: ["/scenarios/experience-manager/pages/fleet/evidence"],
    });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.evidence.verdictList)).toHaveTextContent(
        "summary-before-table",
      ),
    );
    expect(screen.getByTestId(selectors.experience.evidence.captureImage)).toHaveAttribute(
      "src",
      "/artifacts/capture.png",
    );
    expect(screen.getByTestId(selectors.experience.evidence.treePanel)).toHaveTextContent("Experience depth");

    const evidenceLinks = screen.getAllByTestId(selectors.experience.evidence.evidenceLink);
    expect(evidenceLinks[1]).toBeDefined();
    await userEvent.click(evidenceLinks[1]!);

    expect(screen.getByTestId(selectors.experience.evidence.treePanel)).toHaveTextContent("Experience debt");
  });

  it("renders loading state while evidence is pending", () => {
    mocks.fetchEvidence.mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(<EvidencePage />, {
      routerEntries: ["/scenarios/experience-manager/pages/fleet/evidence"],
    });

    expect(screen.getByTestId(selectors.experience.evidence.captureImage)).toHaveTextContent(
      strings.experience.evidence.loadingEvidence,
    );
    expect(screen.getByTestId(selectors.experience.evidence.verdictList)).toHaveTextContent(
      strings.experience.evidence.loadingEvidence,
    );
  });

  it("renders empty state when no evidence has been persisted", async () => {
    mocks.fetchEvidence.mockResolvedValueOnce([]);

    renderWithProviders(<EvidencePage />, {
      routerEntries: ["/scenarios/experience-manager/pages/fleet/evidence"],
    });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.evidence.verdictList)).toHaveTextContent(
        strings.experience.evidence.emptyVerdicts,
      ),
    );
    expect(screen.getByTestId(selectors.experience.evidence.captureImage)).toHaveTextContent(
      strings.experience.evidence.emptyEvidence,
    );
  });

  it("renders error state when evidence loading fails", async () => {
    mocks.fetchEvidence.mockRejectedValueOnce(new Error("server unavailable"));

    renderWithProviders(<EvidencePage />, {
      routerEntries: ["/scenarios/experience-manager/pages/fleet/evidence"],
    });

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(strings.experience.evidence.loadError),
    );
  });

  it("recaptures by running validation and refreshing evidence", async () => {
    mocks.fetchEvidence
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([
        {
          id: "row-1",
          scenario: "experience-manager",
          page: "fleet",
          route: "/",
          state: "default",
          claim: "debt-table-perceivable",
          claimType: "element-present",
          verdict: "passed",
          captureRef: "scenario=experience-manager,path=/",
          axNodeJson: "{}",
          message: "capture unavailable",
          checkedAt: new Date().toISOString(),
        },
      ]);
    mocks.recaptureScenario.mockResolvedValueOnce({});

    renderWithProviders(<EvidencePage />, {
      routerEntries: ["/scenarios/experience-manager/pages/fleet/evidence"],
    });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.evidence.verdictList)).toHaveTextContent(
        strings.experience.evidence.emptyVerdicts,
      ),
    );
    await userEvent.click(screen.getByTestId(selectors.experience.evidence.recaptureAction));

    await waitFor(() => expect(mocks.recaptureScenario).toHaveBeenCalledWith("experience-manager"));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.evidence.verdictList)).toHaveTextContent(
        "debt-table-perceivable",
      ),
    );
  });
});
