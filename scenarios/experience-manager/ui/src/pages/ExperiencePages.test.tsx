import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { EvidencePage, FindingsPage, FleetPage, ScenarioExplorerPage, StudioPage } from "./ExperiencePages";

const mocks = vi.hoisted(() => ({
  applyFindingsFixes: vi.fn(),
  applyStudioDraft: vi.fn(),
  compareStudioVariants: vi.fn(),
  fetchEvidence: vi.fn(),
  fetchFindings: vi.fn(),
  fetchProviderValidation: vi.fn().mockResolvedValue({
    assessment: { presentation: { contractVersion: "v1", capabilities: [] } },
  }),
  fetchFleet: vi.fn(),
  fetchScenarioSpec: vi.fn(),
  previewFindingsFixes: vi.fn(),
  promoteStudioVariant: vi.fn(),
  recaptureScenario: vi.fn(),
  renderStudioSpec: vi.fn(),
  suggestStudioBindings: vi.fn(),
}));

vi.mock("../api/experience", async () => {
  const actual = await vi.importActual<typeof import("../api/experience")>("../api/experience");
  return {
    ...actual,
    applyFindingsFixes: mocks.applyFindingsFixes,
    applyStudioDraft: mocks.applyStudioDraft,
    compareStudioVariants: mocks.compareStudioVariants,
    fetchEvidence: mocks.fetchEvidence,
    fetchFindings: mocks.fetchFindings,
    fetchProviderValidation: mocks.fetchProviderValidation,
    fetchFleet: mocks.fetchFleet,
    fetchScenarioSpec: mocks.fetchScenarioSpec,
    previewFindingsFixes: mocks.previewFindingsFixes,
    promoteStudioVariant: mocks.promoteStudioVariant,
    recaptureScenario: mocks.recaptureScenario,
    renderStudioSpec: mocks.renderStudioSpec,
    suggestStudioBindings: mocks.suggestStudioBindings,
  };
});

function studioSpecFixture() {
  return [
    {
      document: { id: "fleet", title: "Fleet", path: "pages/fleet.json", status: "active" },
      spec: {
        page: {
          id: "fleet",
          title: "Fleet",
          routes: ["/"],
          purpose: "Fleet overview",
          prd_refs: ["OT-P0-001"],
        },
        priorities: [{ statement: "Coverage first." }],
        states: [{ id: "default", description: "Default." }],
        elements: [{ id: "debt-table", role: "table", name: "Experience debt" }],
        claims: [
          {
            id: "debt-table-perceivable",
            type: "element-present",
            tier: "machine",
            statement: "The debt table is perceivable.",
            elements: ["debt-table"],
            states: ["default"],
          },
        ],
      },
    },
  ];
}

describe("FleetPage", () => {
  afterEach(() => {
    cleanup();
    vi.resetAllMocks();
  });

  it("renders live fleet data without fallback rows", async () => {
    mocks.fetchFleet.mockResolvedValueOnce({
      scenarios: [
        {
          scenario: "experience-manager",
          hasExperience: true,
          maxDepth: "L3",
          maxDepthValue: 3,
          pageCount: 5,
          findingCount: 2,
          debtScore: 10,
          status: "findings",
        },
        {
          scenario: "business-health",
          hasExperience: true,
          maxDepth: "L2",
          maxDepthValue: 2,
          pageCount: 3,
          findingCount: 0,
          debtScore: 0,
          status: "clean",
        },
      ],
      scenarioCount: 4,
      withExperienceCount: 1,
      totalPages: 5,
    });

    renderWithProviders(<FleetPage />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.fleet.debtTable)).toHaveTextContent("experience-manager"),
    );
    expect(screen.getByTestId(selectors.experience.fleet.debtTable)).toHaveTextContent("L3");
    expect(screen.getByTestId(selectors.experience.fleet.debtTable)).toHaveTextContent("business-health");
    const table = screen.getByTestId(selectors.experience.fleet.debtTable);
    expect(table).toHaveAttribute("data-testid", selectors.experience.fleet.debtTable);
    expect(table.querySelector('th[aria-sort="ascending"]')).not.toBeNull();
    expect(screen.getByRole("button", { name: strings.experience.fleet.filterAll })).toHaveAttribute("aria-pressed", "true");

    await userEvent.click(screen.getByRole("button", { name: strings.experience.fleet.filterFindings }));
    expect(table).toHaveTextContent("experience-manager");
    expect(table).not.toHaveTextContent("business-health");
    expect(screen.getByRole("button", { name: strings.experience.fleet.filterFindings })).toHaveAttribute("aria-pressed", "true");

    await userEvent.click(screen.getByRole("button", { name: strings.experience.fleet.filterClean }));
    expect(table).not.toHaveTextContent("experience-manager");
    expect(table).toHaveTextContent("business-health");
    expect(screen.getByRole("button", { name: strings.experience.fleet.filterClean })).toHaveAttribute("aria-pressed", "true");
  });

  it("renders loading, empty, and error states from the real query", async () => {
    mocks.fetchFleet.mockReturnValueOnce(new Promise(() => {}));
    const { unmount } = renderWithProviders(<FleetPage />);
    expect(screen.getByTestId(selectors.experience.fleet.debtTable)).toHaveTextContent(
      strings.experience.fleet.loadingData,
    );

    unmount();
    cleanup();
    vi.resetAllMocks();
    mocks.fetchFleet.mockResolvedValueOnce({
      scenarios: [],
      scenarioCount: 0,
      withExperienceCount: 0,
      totalPages: 0,
    });
    renderWithProviders(<FleetPage />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.fleet.debtTable)).toHaveTextContent(
        strings.experience.fleet.emptyFleet,
      ),
    );

    cleanup();
    vi.resetAllMocks();
    mocks.fetchFleet.mockRejectedValueOnce(new Error("server unavailable"));
    renderWithProviders(<FleetPage />);
    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(strings.experience.fleet.loadError),
    );
  });
});

describe("ScenarioExplorerPage", () => {
  afterEach(() => {
    cleanup();
    vi.resetAllMocks();
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

describe("FindingsPage", () => {
  afterEach(() => {
    cleanup();
    vi.resetAllMocks();
  });

  it("renders live validation findings and previews deterministic fixes", async () => {
    mocks.fetchFindings.mockResolvedValueOnce([
      {
        code: "experience.binding_unresolved",
        severity: "warning",
        message: "debt table has no binding",
        location: "experience/pages/fleet.json",
        remediation: "Add a binding",
      },
    ]);
    mocks.previewFindingsFixes.mockResolvedValueOnce({
      scenario: "experience-manager",
      applied: false,
      candidates: [
        {
          ruleId: "experience-fix.binding_drift_repair",
          filePath: "experience/pages/fleet.json",
          description: "Add deterministic placeholder bindings",
          before: "{}\n",
          after: "{\"bindings\":{}}\n",
          applied: false,
        },
      ],
      messages: [],
    });

    renderWithProviders(<FindingsPage />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.findings.findingsList)).toHaveTextContent(
        "experience.binding_unresolved",
      ),
    );
    await userEvent.click(screen.getByTestId(selectors.experience.findings.previewAction));

    await waitFor(() => expect(mocks.previewFindingsFixes).toHaveBeenCalledWith("experience-manager"));
    expect(screen.getByText(strings.experience.findings.fixPreviewLabel)).toBeInTheDocument();
    expect(screen.getByText(/experience-fix\.binding_drift_repair/)).toBeInTheDocument();
  });

  it("applies only after a preview supplies rule ids and refreshes findings", async () => {
    mocks.fetchFindings
      .mockResolvedValueOnce([
        {
          code: "experience.index_parity",
          severity: "error",
          message: "page missing from index",
          location: "experience/index.json",
        },
      ])
      .mockResolvedValueOnce([]);
    mocks.previewFindingsFixes.mockResolvedValueOnce({
      scenario: "experience-manager",
      applied: false,
      candidates: [
        {
          ruleId: "experience-fix.index_normalization",
          filePath: "experience/index.json",
          description: "Normalize index",
          before: "{}",
          after: "{\"pages\":[]}",
          applied: false,
        },
      ],
      messages: [],
    });
    mocks.applyFindingsFixes.mockResolvedValueOnce({
      preview: {
        scenario: "experience-manager",
        applied: true,
        candidates: [
          {
            ruleId: "experience-fix.index_normalization",
            filePath: "experience/index.json",
            description: "Normalize index",
            before: "{}",
            after: "{\"pages\":[]}",
            applied: true,
          },
        ],
        messages: [],
      },
      validation: { report: { findings: [] } },
    });

    renderWithProviders(<FindingsPage />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.findings.findingsList)).toHaveTextContent(
        "experience.index_parity",
      ),
    );
    await userEvent.click(screen.getByTestId(selectors.experience.findings.previewAction));
    await waitFor(() => expect(screen.getByTestId(selectors.experience.findings.applyAction)).toBeEnabled());
    await userEvent.click(screen.getByTestId(selectors.experience.findings.applyAction));

    await waitFor(() =>
      expect(mocks.applyFindingsFixes).toHaveBeenCalledWith("experience-manager", [
        "experience-fix.index_normalization",
      ]),
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.findings.findingsList)).toHaveTextContent(
        strings.experience.findings.emptyFindings,
      ),
    );
  });

  it("renders loading, empty, and error states for findings", async () => {
    mocks.fetchFindings.mockReturnValueOnce(new Promise(() => {}));
    const { unmount } = renderWithProviders(<FindingsPage />);
    expect(screen.getByTestId(selectors.experience.findings.findingsList)).toHaveTextContent(
      strings.experience.findings.loadingFindings,
    );

    unmount();
    cleanup();
    vi.resetAllMocks();
    mocks.fetchFindings.mockResolvedValueOnce([]);
    renderWithProviders(<FindingsPage />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.findings.findingsList)).toHaveTextContent(
        strings.experience.findings.emptyFindings,
      ),
    );

    cleanup();
    vi.resetAllMocks();
    mocks.fetchFindings.mockRejectedValueOnce(new Error("server unavailable"));
    renderWithProviders(<FindingsPage />);
    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(strings.experience.findings.loadError),
    );
  });
});

describe("EvidencePage", () => {
  afterEach(() => {
    cleanup();
    vi.resetAllMocks();
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
        checkedAt: "2026-07-06T10:00:00Z",
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
        checkedAt: "2026-07-06T09:00:00Z",
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

describe("StudioPage", () => {
  afterEach(() => {
    cleanup();
    vi.resetAllMocks();
  });

  it("renders live spec data, backend preview, variants, and binding suggestions", async () => {
    mocks.fetchScenarioSpec.mockResolvedValueOnce(studioSpecFixture());
    mocks.renderStudioSpec.mockResolvedValueOnce({ html: "<section>Rendered fleet wireframe</section>" });
    mocks.compareStudioVariants.mockResolvedValueOnce({
      html: "<section>Draft variant preview</section>",
      degradedReason: "",
      variants: [
        { id: "draft", title: "Fleet draft", html: "<section>Draft</section>" },
        { id: "evidence-forward", title: "Evidence-forward", html: "<section>Evidence</section>" },
      ],
    });
    mocks.suggestStudioBindings.mockResolvedValueOnce([
      {
        elementId: "debt-table",
        testid: "fleet-debt-table",
        role: "table",
        accessibleName: "Experience debt",
        source: "spec",
      },
    ]);

    renderWithProviders(<StudioPage />, {
      routerEntries: ["/studio/experience-manager/fleet"],
    });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.studio.specForm)).toHaveTextContent("Fleet"),
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.studio.wireframePreview)).toHaveTextContent(
        "Draft variant preview",
      ),
    );
    expect(screen.getByTestId(selectors.experience.studio.variantRail)).toHaveTextContent("Fleet draft");
    expect(screen.getByTestId(selectors.experience.studio.specForm)).toHaveTextContent("fleet-debt-table");
  });

  it("applies the current draft through the Studio session flow", async () => {
    mocks.fetchScenarioSpec.mockResolvedValueOnce(studioSpecFixture());
    mocks.renderStudioSpec.mockResolvedValueOnce({ html: "<section>Rendered fleet wireframe</section>" });
    mocks.compareStudioVariants.mockResolvedValue({ html: "<section>Draft</section>", degradedReason: "", variants: [] });
    mocks.suggestStudioBindings.mockResolvedValueOnce([]);
    mocks.applyStudioDraft.mockResolvedValueOnce({
      applied: true,
      diffs: [{ path: "experience/pages/fleet.json", action: "update", before: "old", after: "new" }],
      validation: { report: { findings: [] } },
    });

    renderWithProviders(<StudioPage />, {
      routerEntries: ["/studio/experience-manager/fleet"],
    });

    await waitFor(() => expect(mocks.compareStudioVariants).toHaveBeenCalled());
    const titleInput = await screen.findByLabelText(strings.experience.studio.pageTitle);
    fireEvent.change(titleInput, { target: { value: "Fleet cockpit" } });
    await waitFor(() => expect(titleInput).toHaveValue("Fleet cockpit"));
    await userEvent.click(screen.getByTestId(selectors.experience.studio.saveAction));

    await waitFor(() => expect(mocks.applyStudioDraft).toHaveBeenCalled());
    expect(mocks.applyStudioDraft.mock.calls[0]?.[0]).toBe("experience-manager");
    expect(mocks.applyStudioDraft.mock.calls[0]?.[1]).toMatchObject({
      id: "fleet",
      title: "Fleet cockpit",
      claims: [{ id: "debt-table-perceivable" }],
    });
    expect(screen.getByTestId(selectors.experience.studio.validationSummary)).toHaveTextContent(
      strings.experience.studio.validationCopy,
    );
  });

  it("shows parser findings and does not claim a clean draft when apply returns errors", async () => {
    mocks.fetchScenarioSpec.mockResolvedValueOnce(studioSpecFixture());
    mocks.renderStudioSpec.mockResolvedValueOnce({ html: "<section>Rendered fleet wireframe</section>" });
    mocks.compareStudioVariants.mockResolvedValueOnce({ html: "<section>Draft</section>", degradedReason: "", variants: [] });
    mocks.suggestStudioBindings.mockResolvedValueOnce([]);
    mocks.applyStudioDraft.mockResolvedValueOnce({
      applied: false,
      diffs: [],
      validation: {
        report: {
          findings: [{ severity: "error", title: "Page id is missing" }],
        },
      },
    });

    renderWithProviders(<StudioPage />, {
      routerEntries: ["/studio/experience-manager/fleet"],
    });

    await screen.findByLabelText(strings.experience.studio.pageTitle);
    await userEvent.click(screen.getByTestId(selectors.experience.studio.saveAction));

    await waitFor(() => expect(mocks.applyStudioDraft).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.studio.validationSummary)).toHaveTextContent(
        "error: Page id is missing",
      ),
    );
  });

  it("promotes the current variant through the backend", async () => {
    mocks.fetchScenarioSpec.mockResolvedValueOnce(studioSpecFixture());
    mocks.renderStudioSpec.mockResolvedValueOnce({ html: "<section>Rendered fleet wireframe</section>" });
    mocks.compareStudioVariants.mockResolvedValueOnce({
      html: "<section>Draft</section>",
      degradedReason: "",
      variants: [{ id: "draft", title: "Fleet draft", html: "<section>Draft</section>" }],
    });
    mocks.suggestStudioBindings.mockResolvedValueOnce([]);
    mocks.promoteStudioVariant.mockResolvedValueOnce({
      applied: true,
      diffs: [],
      validation: { report: { findings: [] } },
    });

    renderWithProviders(<StudioPage />, {
      routerEntries: ["/studio/experience-manager/fleet"],
    });

    await screen.findByLabelText(strings.experience.studio.pageTitle);
    await userEvent.click(screen.getByTestId(selectors.experience.studio.promoteAction));

    await waitFor(() => expect(mocks.promoteStudioVariant).toHaveBeenCalled());
    expect(mocks.promoteStudioVariant.mock.calls[0]?.[0]).toBe("experience-manager");
    expect(mocks.promoteStudioVariant.mock.calls[0]?.[1]).toBe("fleet");
    expect(mocks.promoteStudioVariant.mock.calls[0]?.[2]).toMatchObject({ id: "draft" });
  });

  it("renders an error state when Studio spec loading fails", async () => {
    mocks.fetchScenarioSpec.mockRejectedValueOnce(new Error("server unavailable"));
    mocks.renderStudioSpec.mockResolvedValueOnce({ html: "" });
    mocks.compareStudioVariants.mockResolvedValueOnce({ html: "", degradedReason: "", variants: [] });
    mocks.suggestStudioBindings.mockResolvedValueOnce([]);

    renderWithProviders(<StudioPage />, {
      routerEntries: ["/studio/experience-manager/fleet"],
    });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.experience.studio.validationSummary)).toHaveTextContent(
        strings.experience.studio.loadError,
      ),
    );
  });
});
