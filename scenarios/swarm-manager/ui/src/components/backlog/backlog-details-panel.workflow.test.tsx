// [REQ:REQ-P1-002-UI-OPERATIONS-PARITY]
/**
 * Integration-style tests: backlog details surface driven by a mocked
 * canonical workflow projection — migration banner, legacy (pre-migration)
 * affordance, and header CTAs sourced from the projection's legal_actions.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { BacklogDetailsPanel } from "./backlog-details-panel";
import { HeaderPrimaryAction } from "./header-primary-action";
import { BacklogDetailProvider } from "../../contexts/BacklogDetailContext";
import { agentOperationsService } from "../../services";
import { applyWorkflowLegalActions, getItemActions } from "../../lib";
import type { BacklogItem } from "../../types";
import type { WorkflowMigrationStatus, WorkflowProjection } from "../../types/agent-operations";

vi.mock("../../services", () => ({
  agentOperationsService: {
    getMigrationStatus: vi.fn(),
    getWorkflowProjection: vi.fn(),
    listExecutionHistory: vi.fn(),
  },
}));

const mockedMigration = vi.mocked(agentOperationsService.getMigrationStatus);

const makeItem = (overrides?: Partial<BacklogItem>): BacklogItem => ({
  name: "test-item",
  title: "Test Item",
  description: "Test description",
  kind: "execute",
  status: "ready",
  priority: 2,
  tags: [],
  suggestedSkills: [],
  created: "2026-03-20T00:00:00Z",
  updated: "2026-03-20T00:00:00Z",
  ...overrides,
});

function makeProjection(overrides: Partial<WorkflowProjection> = {}): WorkflowProjection {
  return {
    found: true,
    instanceId: "wf-1",
    domainKind: "backlog-item",
    domainId: "execute/test-item",
    state: "open",
    version: 1,
    operations: [],
    decisions: [],
    timers: [],
    legalActions: [],
    policyId: "",
    policyRevision: "",
    ...overrides,
  };
}

const steadyMigration: WorkflowMigrationStatus = {
  state: "not-started",
  epoch: 0,
  stagedCount: 0,
  promotedCount: 0,
  quarantinedCount: 0,
  startedAt: "",
  updatedAt: "",
  reportPath: "",
  documentFound: false,
};

function renderPanel(
  projection: WorkflowProjection | undefined,
  options?: { workflowProjectionError?: boolean },
) {
  return renderWithProviders(
    <BacklogDetailsPanel
      item={makeItem()}
      depRelations={{ parents: [], children: [] }}
      spawnedItems={undefined}
      isLocked={false}
      onEditGlobs={() => undefined}
      onDepStatusChange={() => undefined}
      onSaveNote={async () => undefined}
      workflowProjection={projection}
      workflowProjectionError={options?.workflowProjectionError}
    />,
  );
}

describe("BacklogDetailsPanel — canonical workflow surfaces", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedMigration.mockResolvedValue(steadyMigration);
  });

  it("shows the legacy (pre-migration) affordance when no workflow document exists", async () => {
    renderPanel(makeProjection({ found: false }));
    expect(await screen.findByTestId("no-workflow-notice")).toHaveTextContent(
      "No workflow yet",
    );
  });

  it("shows no legacy affordance when the canonical workflow exists", async () => {
    renderPanel(makeProjection({ found: true }));
    await waitFor(() => expect(mockedMigration).toHaveBeenCalled());
    expect(screen.queryByTestId("no-workflow-notice")).not.toBeInTheDocument();
  });

  it("shows no legacy affordance while the projection has not resolved", async () => {
    renderPanel(undefined);
    await waitFor(() => expect(mockedMigration).toHaveBeenCalled());
    expect(screen.queryByTestId("no-workflow-notice")).not.toBeInTheDocument();
  });

  it("renders the migration banner inside the panel while staged", async () => {
    mockedMigration.mockResolvedValue({
      ...steadyMigration,
      state: "staged",
      stagedCount: 4,
      epoch: 1,
      documentFound: true,
    });
    renderPanel(makeProjection());
    expect(await screen.findByTestId("workflow-migration-banner")).toHaveTextContent(
      "Workflow migration staged",
    );
  });

  it("renders the quarantined migration warning inside the panel", async () => {
    mockedMigration.mockResolvedValue({
      ...steadyMigration,
      state: "quarantined",
      quarantinedCount: 2,
      epoch: 1,
      documentFound: true,
    });
    renderPanel(makeProjection());
    const banner = await screen.findByTestId("workflow-migration-banner");
    expect(banner).toHaveTextContent("Workflow migration quarantined");
    expect(banner).toHaveTextContent("Epoch 1: 2 documents quarantined");
  });

  it("shows an honest fallback notice when the projection query fails", async () => {
    renderPanel(undefined, { workflowProjectionError: true });
    const notice = await screen.findByTestId("workflow-projection-error-notice");
    expect(notice).toHaveTextContent(/legacy pipeline/i);
    // A query failure is not a legacy item — no pre-migration badge.
    expect(screen.queryByTestId("no-workflow-notice")).not.toBeInTheDocument();
  });

  it("shows no projection-error notice in the healthy path", async () => {
    renderPanel(makeProjection());
    await waitFor(() => expect(mockedMigration).toHaveBeenCalled());
    expect(screen.queryByTestId("workflow-projection-error-notice")).not.toBeInTheDocument();
  });
});

describe("Header primary action — canonical legal actions drive the CTA", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedMigration.mockResolvedValue(steadyMigration);
  });

  function renderHeader(projection: WorkflowProjection | null) {
    const item = makeItem();
    // Same composition as useBacklogDetailData: client funnel, then the
    // canonical gate when the projection is present.
    const clientActions = getItemActions({
      item: { kind: item.kind, name: item.name, status: item.status, dependsOn: [] },
      blockingInfo: null,
      readinessReady: true,
      pendingSynthesis: false,
      agentRunning: false,
      hasPendingDecisions: false,
      hasExecutionHistory: false,
    });
    const itemActions = applyWorkflowLegalActions(clientActions, projection);
    return renderWithProviders(
      <BacklogDetailProvider
        value={{
          backlogKind: item.kind,
          name: item.name,
          item,
          itemActions,
          isLocked: false,
          isTerminal: false,
          agentRunIsActive: false,
          latestAgentActivity: null,
          deliverableLabel: "Plan",
          workshopActionLabel: "Workshop",
          agentRunningLabel: "Agent running…",
          agentLabel: "Workshop",
          isWorkshopFinalized: false,
          workshopBlockedDeps: [],
          isRunningAgent: false,
          workshopAutoAdvance: null,
          clearWorkshopAutoAdvance: () => undefined,
        }}
      >
        <HeaderPrimaryAction onFinalizeWorkshop={() => undefined} onRunWorkshop={() => undefined} />
      </BacklogDetailProvider>,
    );
  }

  it("renders the canonical CTA (workshop) when the projection only allows a workshop round", () => {
    renderHeader(makeProjection({ legalActions: ["commit-workshop-round"] }));
    expect(screen.getByRole("button", { name: /workshop/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /run/i })).not.toBeInTheDocument();
  });

  it("falls back to the legacy CTA (run) when no workflow document exists", () => {
    renderHeader(makeProjection({ found: false, legalActions: [] }));
    expect(screen.getByRole("button", { name: /run/i })).toBeInTheDocument();
  });

  it("falls back to the legacy CTA when the projection query errored (null gate rule)", () => {
    // A failed projection query yields no gate — the documented rule keeps
    // the client funnel unchanged rather than freezing the controls.
    renderHeader(null);
    expect(screen.getByRole("button", { name: /run/i })).toBeInTheDocument();
  });
});
