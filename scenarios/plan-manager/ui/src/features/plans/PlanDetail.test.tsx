/**
 * PlanDetail tests — detail rendering, not-found empty state, markdown toggle,
 * and axe-clean structure. The api/plans boundary is mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import {
  PlanSchema,
  PlanEdgeSchema,
  PlanStatus,
  PhaseSchema,
  PhaseStatus,
  ReferenceSchema,
  RelevantContextItemSchema,
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  RelevantContextStatus,
  RegressionAnchorSchema,
  StalenessTier,
  WorkPosture,
  WorkPostureSource,
  ImportProvenanceSchema,
  LegacySectionSchema,
  PlanDefinitionSchema,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const getPlan = vi.fn();
const getGraph = vi.fn();
const renderPlan = vi.fn();
const archivePlan = vi.fn();

vi.mock("../../api/plans", () => ({
  getPlan: (...a: unknown[]) => getPlan(...a),
  getGraph: (...a: unknown[]) => getGraph(...a),
  renderPlan: (...a: unknown[]) => renderPlan(...a),
  archivePlan: (...a: unknown[]) => archivePlan(...a),
  listPlans: vi.fn(),
  listTemplates: vi.fn(),
  createFromTemplate: vi.fn(),
}));

import { PlanDetail } from "./PlanDetail";

const fullPlan = create(PlanSchema, {
  id: "plan-1",
  slug: "migrate-auth",
  title: "Migrate auth",
  status: PlanStatus.ACTIVE,
  updatedAt: "2026-06-25T10:00:00Z",
  purpose: "Move auth to Connect.",
  scope: "auth scenario only",
  constraints: "No consumer inversion in this pass.",
  nonGoals: "Do not rewrite root planning commands.",
  definitionOfDone: "all green",
  references: [
    create(ReferenceSchema, { id: "r1", target: "api/main.go", staleness: StalenessTier.FRESH }),
  ],
  relevantContext: [
    create(RelevantContextItemSchema, {
      id: "ctx-global",
      kind: RelevantContextKind.SEARCH,
      label: "Recall prior plan work",
      command: "search-hub query plan-manager --type record",
      repeatPolicy: RelevantContextRepeatPolicy.ON_RESUME,
      status: RelevantContextStatus.READY,
    }),
    create(RelevantContextItemSchema, {
      id: "ctx-global-doc",
      kind: RelevantContextKind.DOC,
      target: "docs/concepts/PLAN-MODEL.md",
      reason: "Read model details before changing storage.",
      instruction: "Confirm typed anchors remain canonical.",
      repeatPolicy: RelevantContextRepeatPolicy.AS_NEEDED,
      status: RelevantContextStatus.DEGRADED,
    }),
  ],
  regressionAnchor: create(RegressionAnchorSchema, {
    strategy: "scenario_baseline",
    scenario: "auth",
    commands: ["vrooli scenario test auth"],
  }),
  phases: [
    create(PhaseSchema, {
      id: "p1",
      order: 1,
      title: "Contracts",
      intent: "define proto",
      status: PhaseStatus.DONE,
      acceptance: "proto fields round-trip",
      reminders: ["keep CLI thin"],
      baselineScope: ["packages/proto/schemas/plan-manager/**"],
      relevantContext: [
        create(RelevantContextItemSchema, {
          id: "ctx-phase",
          kind: RelevantContextKind.DOC,
          label: "Plan model",
          target: "scenarios/plan-manager/docs/concepts/PLAN-MODEL.md",
          repeatPolicy: RelevantContextRepeatPolicy.PHASE_ENTRY,
        }),
      ],
    }),
  ],
});

describe("PlanDetail", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the not-found empty state when the plan is absent", async () => {
    getPlan.mockResolvedValue(undefined);
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="missing" />);

    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.pages.planDetail}-${selectors.asyncSuffix.empty}`),
      ).toBeInTheDocument();
    });
  });

  it("renders the plan body with phases and references", async () => {
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="plan-1" />);

    expect(await screen.findByTestId(selectors.pages.planDetail)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plans.phase({ id: "p1" }))).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.plans.relevantContext).length).toBeGreaterThan(0);
  });

  it("renders plan definitions and routes GFM through the markdown renderer", async () => {
    getPlan.mockResolvedValue(create(PlanSchema, {
      ...fullPlan,
      purpose: "| Feature | State |\n|---|---|\n| Markdown | rendered |\n\n```mermaid\nflowchart LR\nA --> B\n```",
      definitions: [create(PlanDefinitionSchema, { term: "Trust gate", meaning: "Required validation checkpoint." })],
    }));
    getGraph.mockResolvedValue([]);
    renderWithProviders(<PlanDetail planId="definitions" />);
    expect(await screen.findByTestId(selectors.pages.planDetail)).toBeInTheDocument();
    expect(screen.getByText("Trust gate")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open" })).toBeInTheDocument();
    expect(document.querySelector("table")).toBeInTheDocument();
  });

  it("renders the professional structured fields and work posture", async () => {
    getPlan.mockResolvedValue(
      create(PlanSchema, {
        id: "plan-pro",
        slug: "pro-plan",
        title: "Professional plan",
        status: PlanStatus.ACTIVE,
        problemStatement: "The model is too thin to review.",
        targetOutcome: "A reviewable rendered plan.",
        assumptions: "Baseline captured first.",
        technicalApproach: "Model-first contract change.",
        prohibitedApproaches: "No legacy cloning.",
        validationStrategy: "Run the suites then the scenario test.",
        finalValidationCommands: ["vrooli scenario test plan-manager"],
        risksHazards: "Too many fields makes it heavy.",
        workPosture: WorkPosture.GREENFIELD,
        workPostureSource: WorkPostureSource.SERVICE_MATURITY,
        workPostureDetail: 'Scenario "plan-manager" maturity is greenfield.',
        importProvenance: create(ImportProvenanceSchema, {
          sourcePath: "docs/plans/old.md",
          originalFormat: "legacy_markdown",
        }),
        preservedLegacySections: [
          create(LegacySectionSchema, {
            heading: "Contract Decisions",
            content: "REST stays for now.",
            preservationReason: "unmapped_legacy_section",
          }),
        ],
        phases: [
          create(PhaseSchema, {
            id: "pp1",
            order: 1,
            title: "Contract",
            intent: "lock the model",
            status: PhaseStatus.TODO,
            affectedAreas: ["model.proto"],
            steps: ["Add proto fields", "Regenerate"],
            expectedOutputs: ["Generated code compiles"],
            validation: "go test ./internal/planproto",
            acceptance: "round-trips",
            risksHazards: ["field churn"],
            handoffNotes: "phase 2 depends on this",
          }),
        ],
      }),
    );
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="plan-pro" />);

    expect(await screen.findByTestId(selectors.pages.planDetail)).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.problemHeading))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.targetOutcomeHeading))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.technicalApproachHeading))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.validationStrategyHeading))).toBeInTheDocument();
    expect(screen.getAllByText(i18n.t(strings.pages.plans.detail.workPostureHeading)).length).toBeGreaterThan(0);
    // Data values (from the plan, not copy) — matched by regex so the no-string-
    // literal copy lint rule (which targets i18n copy) is satisfied.
    expect(screen.getByText(/^greenfield$/)).toBeInTheDocument();
    expect(screen.getByText(/maturity is greenfield/)).toBeInTheDocument();
    // Phase professional fields render.
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.phaseSteps))).toBeInTheDocument();
    expect(screen.getByText(/Add proto fields/)).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.phaseValidation))).toBeInTheDocument();
    // Import provenance + preserved legacy.
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.importProvenanceHeading))).toBeInTheDocument();
    expect(screen.getByText(/Contract Decisions/)).toBeInTheDocument();
  });

  it("renders empty optional plan sections without placeholders leaking", async () => {
    getPlan.mockResolvedValue(create(PlanSchema, { id: "plan-empty", title: "Empty plan" }));
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="plan-empty" />);

    expect(await screen.findByTestId(selectors.pages.planDetail)).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.noPhases))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.noReferences))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.anchorNone))).toBeInTheDocument();
    expect(await screen.findByText(i18n.t(strings.pages.plans.detail.edgeNone))).toBeInTheDocument();
  });

  it("renders legacy phase setup and complete regression anchor fields", async () => {
    getPlan.mockResolvedValue(create(PlanSchema, {
      ...fullPlan,
      id: "plan-legacy",
      relevantContext: [],
      regressionAnchor: create(RegressionAnchorSchema, {
        strategy: "sha_allowlist",
        scenario: "plan-manager",
        baselineName: "plan-manager-hardening-readiness",
        headSha: "abc123",
        allowlistPaths: ["scenarios/plan-manager/**"],
        commands: ["git diff --stat abc123 -- scenarios/plan-manager"],
        unavailable: true,
      }),
      phases: [
        create(PhaseSchema, {
          id: "legacy-phase",
          order: 1,
          title: "Legacy setup",
          status: PhaseStatus.TODO,
          acceptance: "legacy still renders",
          requiredReading: ["docs/legacy-required-reading.md"],
          reminders: ["NO_CONTEXT: migrated fixture"],
          baselineScope: ["scenarios/plan-manager/ui/**"],
        }),
      ],
    }));
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="plan-legacy" />);

    const page = await screen.findByTestId(selectors.pages.planDetail);
    expect(page).toHaveTextContent("docs/legacy-required-reading.md");
    expect(page).toHaveTextContent("NO_CONTEXT: migrated fixture");
    expect(page).toHaveTextContent("scenarios/plan-manager/ui/**");
    expect(page).toHaveTextContent("plan-manager-hardening-readiness");
    expect(page).toHaveTextContent("abc123");
    expect(page).toHaveTextContent("scenarios/plan-manager/**");
    expect(page).toHaveTextContent("git diff --stat abc123 -- scenarios/plan-manager");
    expect(page).toHaveTextContent(i18n.t(strings.pages.plans.detail.anchorUnavailable));
  });

  it("renders supersession graph directions", async () => {
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([
      create(PlanEdgeSchema, { fromPlanId: "plan-1", toPlanId: "old-plan", kind: "supersedes" }),
      create(PlanEdgeSchema, { fromPlanId: "new-plan", toPlanId: "plan-1", kind: "supersedes" }),
    ]);

    renderWithProviders(<PlanDetail planId="plan-1" />);

    const graph = await screen.findByTestId(selectors.plans.detailGraph);
    expect(graph.textContent).toContain("old-plan");
    expect(graph.textContent).toContain("new-plan");
  });

  it("lazily fetches and shows rendered markdown when toggled", async () => {
    const user = userEvent.setup();
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([]);
    renderPlan.mockResolvedValue("# Migrate auth");

    renderWithProviders(<PlanDetail planId="plan-1" />);

    await screen.findByTestId(selectors.pages.planDetail);
    expect(renderPlan).not.toHaveBeenCalled();

    await user.click(screen.getByTestId(selectors.plans.detailMarkdownToggle));
    await waitFor(() => {
      expect(renderPlan).toHaveBeenCalledWith("plan-1");
      expect(screen.getByTestId(selectors.plans.detailMarkdown)).toBeInTheDocument();
    });
  });

  it("renders the detail without axe violations", async () => {
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([]);

    const { container } = renderWithProviders(<PlanDetail planId="plan-1" />);
    await screen.findByTestId(selectors.pages.planDetail);
    await expectNoA11yViolations(container);
  });
});
