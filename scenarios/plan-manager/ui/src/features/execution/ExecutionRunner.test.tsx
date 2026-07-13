/**
 * ExecutionRunner tests — start gate, just-in-time context, transition,
 * decision/finding capture, complete + handoff, and axe-clean structure.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  GuidedStepSchema,
  HandoffSchema,
  LogEntrySchema,
  LogSummarySchema,
  PhaseSchema,
  PhaseStatus,
  RelevantContextItemSchema,
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  RelevantContextStatus,
  StalenessTier,
  ValidationResultSchema,
  ValidationVerdict,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import {
  CompletionNudgeSchema,
  ExecutionSchema,
  PhaseFeedbackCheckpointSchema,
  PhaseContextSchema,
} from "@vrooli/proto-types/plan-manager/v1/execution/execution_pb";
import { PlanSchema } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const startExecution = vi.fn();
const getStatus = vi.fn();
const getContext = vi.fn();
const transitionPhase = vi.fn();
const completeExecution = vi.fn();
const addDecision = vi.fn();
const addFinding = vi.fn();
const addBug = vi.fn();
const addRecord = vi.fn();
const addNote = vi.fn();
const listPlans = vi.fn();

vi.mock("../../api/execution", () => ({
  startExecution: (...a: unknown[]) => startExecution(...a),
  getStatus: (...a: unknown[]) => getStatus(...a),
  getContext: (...a: unknown[]) => getContext(...a),
  transitionPhase: (...a: unknown[]) => transitionPhase(...a),
  completeExecution: (...a: unknown[]) => completeExecution(...a),
  getNext: vi.fn(),
  getHandoff: vi.fn(),
  getVelocity: vi.fn(),
}));
vi.mock("../../api/log", () => ({
  addDecision: (...a: unknown[]) => addDecision(...a),
  addFinding: (...a: unknown[]) => addFinding(...a),
  addBug: (...a: unknown[]) => addBug(...a),
  addRecord: (...a: unknown[]) => addRecord(...a),
  addNote: (...a: unknown[]) => addNote(...a),
}));
vi.mock("../../api/plans", () => ({
  listPlans: (...a: unknown[]) => listPlans(...a),
  listTemplates: vi.fn(),
  getPlan: vi.fn(),
  getGraph: vi.fn(),
  renderPlan: vi.fn(),
  archivePlan: vi.fn(),
  createFromTemplate: vi.fn(),
}));

import { ExecutionRunner } from "./ExecutionRunner";

const execution = create(ExecutionSchema, { id: "exec-1", planId: "plan-1", currentPhaseId: "p1" });
const step = create(GuidedStepSchema, {
  stepKind: "execution_context",
  title: "Review phase context",
  summary: "Use the returned context before transitioning phases.",
  nextActions: [{ label: "Mark done", argv: ["exec", "transition", "exec-1", "p1", "--status", "done"] }],
});
const context = create(PhaseContextSchema, {
  currentPhase: create(PhaseSchema, {
    id: "p1",
    order: 1,
    title: "Contracts",
    intent: "Lock the execution handoff contract.",
    status: PhaseStatus.ACTIVE,
  }),
  nextPhase: create(PhaseSchema, { id: "p2", order: 2, title: "Validate", status: PhaseStatus.TODO }),
  requiredReading: ["docs/PLAN.md"],
  reminders: ["keep it small"],
  relevantContext: [
    create(RelevantContextItemSchema, {
      id: "ctx-1",
      kind: RelevantContextKind.COMMAND,
      label: "Recall",
      command: "search-hub query plan-manager --type record",
      repeatPolicy: RelevantContextRepeatPolicy.ON_RESUME,
    }),
    create(RelevantContextItemSchema, {
      id: "ctx-2",
      kind: RelevantContextKind.DOC,
      label: "",
      target: "docs/context.md",
      reason: "Read when validation context changes.",
      instruction: "Check the document before editing.",
      repeatPolicy: RelevantContextRepeatPolicy.AS_NEEDED,
      status: RelevantContextStatus.DEGRADED,
    }),
  ],
  lastValidation: create(ValidationResultSchema, {
    id: "val-fresh",
    verdict: ValidationVerdict.PASS,
    staleness: StalenessTier.FRESH,
  }),
  staleness: StalenessTier.FRESH,
  inputsFreshened: true,
  freshenStatus: "captured",
  freshenDetail: "captured baseline plan-1-baseline; staleness: 2 reference(s), overall=fresh",
  logSummary: create(LogSummarySchema, {
    total: 3,
    decisions: 2,
    findings: 1,
    candidateFindings: 1,
    pendingSync: 1,
  }),
  feedbackCheckpoint: create(PhaseFeedbackCheckpointSchema, {
    phaseId: "p1",
    reviewed: false,
    satisfied: false,
    summary: "Review phase feedback before marking done.",
    noFeedbackTitle: "Phase feedback reviewed: none",
    noFeedbackDetail: "No decisions, findings, bugs, records, or reusable notes to capture for this phase.",
  }),
});

const startAndLand = async () => {
  const user = userEvent.setup();
  listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
  startExecution.mockResolvedValue({ execution, context, step });
  getStatus.mockResolvedValue({ execution, context, step });
  getContext.mockResolvedValue({ execution, context, step });

  renderWithProviders(<ExecutionRunner />);
  await waitFor(() => {
    expect(
      screen.getByTestId(selectors.execution.planSelect).querySelector('option[value="plan-1"]'),
    ).not.toBeNull();
  });
  await user.selectOptions(screen.getByTestId(selectors.execution.planSelect), "plan-1");
  await user.click(screen.getByTestId(selectors.execution.startButton));
  await screen.findByTestId(selectors.execution.context);
  return user;
};

describe("ExecutionRunner", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("[REQ:PM-EXEC-001] starts a run and renders just-in-time context", async () => {
    await startAndLand();
    expect(startExecution).toHaveBeenCalledWith("plan-1", "");
    expect(screen.getByTestId(selectors.execution.context)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.execution.setupContext)).toHaveTextContent("Recall");
    expect(screen.getByTestId(selectors.execution.guidedStep)).toHaveTextContent("exec transition exec-1 p1 --status done");
    // The phase context surfaces a compact log-ledger roll-up (counts + pending sync).
    expect(screen.getByTestId(selectors.execution.logSummary)).toHaveTextContent("Pending sync");
    expect(screen.getByTestId(selectors.execution.feedbackCheckpoint)).toHaveTextContent(
      "Review phase feedback before marking done.",
    );
    // The execution-start freshen status is surfaced (captured baseline + staleness).
    expect(screen.getByTestId(selectors.execution.freshenStatus)).toHaveTextContent("captured");
  });

  it("renders start errors without leaving the start gate", async () => {
    const user = userEvent.setup();
    listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
    startExecution.mockRejectedValue(new Error("start failed"));

    renderWithProviders(<ExecutionRunner />);
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.execution.planSelect).querySelector('option[value="plan-1"]'),
      ).not.toBeNull();
    });
    await user.selectOptions(screen.getByTestId(selectors.execution.planSelect), "plan-1");
    await user.click(screen.getByTestId(selectors.execution.startButton));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("start failed");
      expect(screen.getByTestId(selectors.execution.startForm)).toBeInTheDocument();
    });
  });

  it("refreshes setup context without advancing execution", async () => {
    const user = await startAndLand();
    await user.click(screen.getByTestId(selectors.execution.contextButton));
    await waitFor(() => {
      expect(getContext).toHaveBeenCalledWith("exec-1");
    });
  });

  it("renders context refresh errors after execution has started", async () => {
    const user = await startAndLand();
    getContext.mockRejectedValueOnce(new Error("refresh failed"));

    await user.click(screen.getByTestId(selectors.execution.contextButton));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("refresh failed");
      expect(screen.getByTestId(selectors.execution.context)).toBeInTheDocument();
    });
  });

  it("transitions the current phase", async () => {
    const user = await startAndLand();
    transitionPhase.mockResolvedValue({ execution, step });
    await user.click(screen.getByTestId(selectors.execution.transitionButton));
    await waitFor(() => {
      expect(transitionPhase).toHaveBeenCalledWith("exec-1", "p1", PhaseStatus.DONE);
    });
  });

  it("records a decision in-flow via the log domain", async () => {
    const user = await startAndLand();
    addDecision.mockResolvedValue({
      entry: create(LogEntrySchema, { id: "d1", title: "chose Connect" }),
      step,
      deduplicated: false,
    });
    await user.type(screen.getByTestId(selectors.execution.decisionSummary), "chose Connect");
    await user.click(screen.getByTestId(selectors.execution.recordDecisionButton));
    await waitFor(() => {
      expect(addDecision).toHaveBeenCalledWith("exec-1", "p1", "chose Connect", { detail: "" });
    });
  });

  it("records a candidate finding in-flow via the log domain", async () => {
    const user = await startAndLand();
    addFinding.mockResolvedValue({
      entry: create(LogEntrySchema, { id: "f1", title: "edge case", detail: "needs triage" }),
      step,
      deduplicated: false,
    });
    await user.type(screen.getByTestId(selectors.execution.findingTitle), "edge case");
    await user.type(screen.getByTestId(selectors.execution.findingDetail), "needs triage");
    await user.click(screen.getByTestId(selectors.execution.recordFindingButton));
    await waitFor(() => {
      expect(addFinding).toHaveBeenCalledWith("exec-1", "p1", "edge case", { detail: "needs triage" });
      expect(screen.getByTestId(selectors.execution.recordFindingButton).closest("section")).toHaveTextContent(
        "edge case",
      );
      expect(screen.getByTestId(selectors.execution.recordFindingButton).closest("section")).toHaveTextContent(
        "needs triage",
      );
    });
  });

  it("captures bug reports, records, notes, and explicit no-feedback checkpoints", async () => {
    const user = await startAndLand();
    addBug.mockResolvedValue({
      entry: create(LogEntrySchema, { id: "b1", title: "confirmed defect", detail: "breaks done gate", capture: { state: "draft", draftId: "bug-1", needs: ["actual"], nextAction: ["prompt-manager", "team", "bug-repair", "scenario-qa", "bug-1"] } }),
      step,
      deduplicated: false,
    });
    addRecord.mockResolvedValue({
      entry: create(LogEntrySchema, { id: "r1", title: "checkpoint pattern", detail: "phase-close review", capture: { state: "published" } }),
      step,
      deduplicated: false,
    });
    addNote.mockResolvedValue({
      entry: create(LogEntrySchema, { id: "n1", title: "Phase feedback reviewed: none" }),
      step,
      deduplicated: false,
    });

    await user.type(screen.getByTestId(selectors.execution.bugTitle), "confirmed defect");
    await user.type(screen.getByTestId(selectors.execution.bugDetail), "breaks done gate");
    await user.type(screen.getByLabelText("Bug signal type"), "regression");
    await user.type(screen.getByLabelText("Bug report severity"), "major");
    await user.type(screen.getByLabelText("Affected scenario"), "plan-manager");
    await user.type(screen.getByLabelText("Bug reproduction steps"), "start, open plan");
    await user.type(screen.getByLabelText("Bug expected behavior"), "fresh data");
    await user.type(screen.getByLabelText("Bug actual behavior"), "stale data");
    await user.type(screen.getByLabelText("Bug taxonomy description"), "details");
    await user.click(screen.getByTestId(selectors.execution.recordBugButton));

    await user.type(screen.getByTestId(selectors.execution.recordTitle), "checkpoint pattern");
    await user.type(screen.getByTestId(selectors.execution.recordDetail), "phase-close review");
    await user.type(screen.getByLabelText("Record kind"), "execute");
    await user.type(screen.getByLabelText("Record scenario"), "plan-manager");
    await user.type(screen.getByLabelText("Record outcome"), "shipped");
    await user.type(screen.getByLabelText("Record trigger"), "close phase");
    await user.type(screen.getByLabelText("Record approach"), "use checkpoint");
    await user.type(screen.getByLabelText("Record evidence"), "go test");
    await user.click(screen.getByTestId(selectors.execution.recordRecordButton));

    await user.type(screen.getByTestId(selectors.execution.noteTitle), "operator note");
    await user.click(screen.getByTestId(selectors.execution.recordNoteButton));

    await user.click(screen.getByTestId(selectors.execution.noFeedbackButton));

    await waitFor(() => {
      expect(addBug).toHaveBeenCalledWith("exec-1", "p1", "confirmed defect", {
        detail: "breaks done gate", signalType: "regression", reportSeverity: "major", repro: ["start", "open plan"],
        expected: "fresh data", actual: "stale data", description: "details", context: { scenario: "plan-manager" }, honestyFlags: [],
      });
      expect(addRecord).toHaveBeenCalledWith("exec-1", "p1", "checkpoint pattern", {
        detail: "phase-close review", kind: "execute", scenario: "plan-manager", trigger: "close phase",
        approach: "use checkpoint", recordEvidence: "go test", outcome: "shipped",
      });
      expect(screen.getByText(/private draft bug-1/i)).toBeInTheDocument();
      expect(addNote).toHaveBeenCalledWith("exec-1", "p1", "operator note", { detail: "" });
      expect(addNote).toHaveBeenCalledWith("exec-1", "p1", "Phase feedback reviewed: none", {
        detail: "No decisions, findings, bugs, records, or reusable notes to capture for this phase.",
      });
      expect(getStatus).toHaveBeenCalled();
    });
  });

  it("renders legacy required reading, no current phase, and last validation fallbacks", async () => {
    const user = userEvent.setup();
    const emptyContext = create(PhaseContextSchema, {
      resumePhaseId: "",
      requiredReading: ["docs/legacy.md"],
      reminders: [],
      lastValidation: create(ValidationResultSchema, {
        id: "val-1",
        verdict: ValidationVerdict.UNKNOWN,
        staleness: StalenessTier.DEFINITELY_STALE,
      }),
      staleness: StalenessTier.DEFINITELY_STALE,
    });
    listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
    startExecution.mockResolvedValue({ execution, context: emptyContext, step });
    getStatus.mockResolvedValue({ execution, context: emptyContext, step });

    renderWithProviders(<ExecutionRunner />);
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.execution.planSelect).querySelector('option[value="plan-1"]'),
      ).not.toBeNull();
    });
    await user.selectOptions(screen.getByTestId(selectors.execution.planSelect), "plan-1");
    await user.click(screen.getByTestId(selectors.execution.startButton));

    const contextPanel = await screen.findByTestId(selectors.execution.context);
    expect(contextPanel).toHaveTextContent("docs/legacy.md");
    expect(contextPanel).toHaveTextContent(/No current phase|No active phase/i);
  });

  it("renders a current phase with no previous validation", async () => {
    const user = userEvent.setup();
    const noValidationContext = create(PhaseContextSchema, {
      currentPhase: create(PhaseSchema, {
        id: "p1",
        order: 1,
        title: "Contracts",
        intent: "Define transition checks.",
        status: PhaseStatus.ACTIVE,
      }),
      staleness: StalenessTier.FRESH,
    });
    listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
    startExecution.mockResolvedValue({ execution, context: noValidationContext, step });

    renderWithProviders(<ExecutionRunner />);
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.execution.planSelect).querySelector('option[value="plan-1"]'),
      ).not.toBeNull();
    });
    await user.selectOptions(screen.getByTestId(selectors.execution.planSelect), "plan-1");
    await user.click(screen.getByTestId(selectors.execution.startButton));

    const contextPanel = await screen.findByTestId(selectors.execution.context);
    expect(contextPanel).toHaveTextContent("Define transition checks.");
    expect(contextPanel).toHaveTextContent("None");
  });

  it("[REQ:PM-HANDOFF-001] completes the run and shows the handoff", async () => {
    const user = await startAndLand();
    completeExecution.mockResolvedValue({
      handoff: create(HandoffSchema, { id: "h1", executionId: "exec-1", staleness: StalenessTier.FRESH }),
      nudges: [],
      step,
    });
    await user.click(screen.getByTestId(selectors.execution.completeButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.execution.handoff)).toBeInTheDocument();
    });
  });

  it("renders completion nudges and optional handoff fields", async () => {
    const user = await startAndLand();
    completeExecution.mockResolvedValue({
      handoff: create(HandoffSchema, {
        id: "h1",
        executionId: "exec-1",
        resumePhaseId: "p2",
        proseHandoffRef: "rec-123",
        staleness: StalenessTier.DEFINITELY_STALE,
        logEntries: [create(LogEntrySchema, { id: "e1", title: "chose Connect" })],
        logSummary: create(LogSummarySchema, { total: 1, decisions: 1 }),
      }),
      nudges: [create(CompletionNudgeSchema, { kind: "record_finding", message: "Record candidate findings" })],
      step,
    });
    await user.click(screen.getByTestId(selectors.execution.completeButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.execution.handoff)).toHaveTextContent("p2");
      expect(screen.getByTestId(selectors.execution.handoff)).toHaveTextContent("rec-123");
      expect(screen.getByTestId(selectors.execution.handoff).closest("section")).toHaveTextContent(
        "Record candidate findings",
      );
      // The handoff surfaces the log-ledger roll-up assembled at completion.
      expect(screen.getByTestId(selectors.execution.handoff).closest("section")).toHaveTextContent("Log entries");
    });
  });

  it("renders the runner without axe violations", async () => {
    listPlans.mockResolvedValue([]);
    const { container } = renderWithProviders(<ExecutionRunner />);
    await screen.findByTestId(selectors.execution.startForm);
    await expectNoA11yViolations(container);
  });
});
