/**
 * AuthoringWizard tests — start gate, section walk + submit (with violations),
 * autofill (degraded honesty), finalize, and axe-clean structure.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  AuthoringSessionSchema,
  AutofillResultSchema,
  ContextCandidateSchema,
  ReferenceCandidateSchema,
  SectionSchema,
  StructureViolationSchema,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";
import { PlanSchema } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import {
  RelevantContextItemSchema,
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  RelevantContextScope,
  ReferenceKind,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const startSession = vi.fn();
const getSession = vi.fn();
const addPhase = vi.fn();
const submitSection = vi.fn();
const submitPhaseField = vi.fn();
const nextPhase = vi.fn();
const validateStructure = vi.fn();
const autofill = vi.fn();
const submitRelevantContextItem = vi.fn();
const updateRelevantContextItem = vi.fn();
const removeRelevantContextItem = vi.fn();
const discoverContextCandidates = vi.fn();
const acceptContextCandidate = vi.fn();
const rejectContextCandidate = vi.fn();
const suggestReferences = vi.fn();
const acceptReferenceCandidate = vi.fn();
const rejectReferenceCandidate = vi.fn();
const finalize = vi.fn();

vi.mock("../../api/authoring", () => ({
  startSession: (...a: unknown[]) => startSession(...a),
  getSession: (...a: unknown[]) => getSession(...a),
  addPhase: (...a: unknown[]) => addPhase(...a),
  submitSection: (...a: unknown[]) => submitSection(...a),
  submitPhaseField: (...a: unknown[]) => submitPhaseField(...a),
  nextPhase: (...a: unknown[]) => nextPhase(...a),
  validateStructure: (...a: unknown[]) => validateStructure(...a),
  autofill: (...a: unknown[]) => autofill(...a),
  submitRelevantContextItem: (...a: unknown[]) => submitRelevantContextItem(...a),
  updateRelevantContextItem: (...a: unknown[]) => updateRelevantContextItem(...a),
  removeRelevantContextItem: (...a: unknown[]) => removeRelevantContextItem(...a),
  discoverContextCandidates: (...a: unknown[]) => discoverContextCandidates(...a),
  acceptContextCandidate: (...a: unknown[]) => acceptContextCandidate(...a),
  rejectContextCandidate: (...a: unknown[]) => rejectContextCandidate(...a),
  suggestReferences: (...a: unknown[]) => suggestReferences(...a),
  acceptReferenceCandidate: (...a: unknown[]) => acceptReferenceCandidate(...a),
  rejectReferenceCandidate: (...a: unknown[]) => rejectReferenceCandidate(...a),
  listRelevantContext: vi.fn(),
  finalize: (...a: unknown[]) => finalize(...a),
  getSection: vi.fn(),
  nextSection: vi.fn(),
}));

import { AuthoringWizard } from "./AuthoringWizard";

const session = create(AuthoringSessionSchema, {
  id: "sess-1",
  title: "New plan",
  planSlug: "new-plan",
  currentSectionKey: "purpose",
  sections: [
    create(SectionSchema, { key: "purpose", label: "Purpose", mandatory: true }),
    create(SectionSchema, { key: "scope", label: "Scope" }),
  ],
});

const step = {
  stepKind: "phase_acceptance",
  title: "Phase acceptance",
  summary: "State the success condition.",
  instructions: ["Make the done state observable."],
  requiredInputs: ["acceptance"],
  examples: [],
  commonMistakes: [],
  nextActions: [{ id: "submit-phase-acceptance", label: "Submit acceptance", argv: ["author", "phase-submit"] }],
};

const phase = {
  id: "phase-1",
  order: 1,
  title: "Authoring contract",
  intent: "Add phase-native UI.",
  references: [],
  requiredReading: [],
  reminders: [],
  acceptance: "",
  noCodeRefsReason: "",
  relevantContext: [],
};

const sessionWithPhase = {
  id: "sess-1",
  title: "New plan",
  planSlug: "new-plan",
  currentSectionKey: "purpose",
  sections: session.sections,
  finalized: false,
  planId: "",
  currentPhaseId: "phase-1",
  relevantContext: [],
  phaseDrafts: [phase],
  contextCandidates: [],
  referenceCandidates: [],
};

const candidateItem = create(RelevantContextItemSchema, {
  id: "ctx-candidate",
  kind: RelevantContextKind.SEARCH,
  label: "Recall records",
  reason: "Prior records reduce duplicate planning work.",
  command: "search-hub query plan-manager --type record",
});

const completePhase = {
  ...phase,
  references: [{ id: "ref-1", target: "api/main.go", kind: ReferenceKind.CODE }],
  relevantContext: [candidateItem],
};

const candidate = create(ContextCandidateSchema, {
  id: "cand-1",
  item: candidateItem,
  concept: "plan-manager context",
  source: "search-hub",
  status: "pending",
});

const referenceCandidate = create(ReferenceCandidateSchema, {
  id: "ref-cand-1",
  reference: { id: "r1", kind: ReferenceKind.CODE, target: "api/internal/authoring/service.go" },
  source: "code-symbol",
  confidence: 0.9,
  status: "pending",
});

describe("AuthoringWizard", () => {
  beforeEach(async () => {
    await setLocale("en");
    // Mutations no longer echo the session; the wizard re-hydrates via getSession
    // (read-after-write). Default to the base session; tests that change state
    // override this with the expected post-mutation session.
    getSession.mockResolvedValue({ session, step });
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("starts a session from the start gate", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });

    renderWithProviders(<AuthoringWizard />);

    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await waitFor(() => {
      expect(startSession).toHaveBeenCalledWith("New plan", "", "");
      expect(screen.getByTestId(selectors.authoring.sections)).toBeInTheDocument();
    });
  });

  it("renders a start-session error at the start gate", async () => {
    const user = userEvent.setup();
    startSession.mockRejectedValue(new Error("cannot start"));

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("cannot start");
    });
  });

  it("surfaces structure violations after submitting a section", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    submitSection.mockResolvedValue({
      session,
      violations: [create(StructureViolationSchema, { sectionKey: "scope", message: "missing" })],
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contentInput);
    await user.type(screen.getByTestId(selectors.authoring.contentInput), "some content");
    await user.click(screen.getByTestId(selectors.authoring.submitButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.authoring.violations)).toBeInTheDocument();
    });
  });

  it("runs structure validation from the current section", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    validateStructure.mockResolvedValue({
      violations: [create(StructureViolationSchema, { sectionKey: "relevant_context", message: "missing context" })],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.validateButton);
    await user.click(screen.getByTestId(selectors.authoring.validateButton));

    await waitFor(() => {
      expect(validateStructure).toHaveBeenCalledWith("sess-1");
      expect(screen.getByTestId(selectors.authoring.violations)).toHaveTextContent("missing context");
    });
  });

  it("renders API errors without losing the session", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    submitSection.mockRejectedValue(new Error("save failed"));

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contentInput);
    await user.click(screen.getByTestId(selectors.authoring.submitButton));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
      expect(screen.getByTestId(selectors.authoring.sections)).toBeInTheDocument();
    });
  });

  it("adds phases and submits phase fields with guidance", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    getSession.mockResolvedValue({ session: sessionWithPhase, step });
    addPhase.mockResolvedValue({
      phase,
      progress: { sessionId: "sess-1", currentPhaseId: "phase-1" },
      violations: [],
      step,
    });
    submitPhaseField.mockResolvedValue({
      phase,
      progress: { sessionId: "sess-1", currentPhaseId: "phase-1" },
      violations: [],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.phaseTitleInput);
    await user.type(screen.getByTestId(selectors.authoring.phaseTitleInput), "Authoring contract");
    await user.type(screen.getByTestId(selectors.authoring.phaseIntentInput), "Add phase-native UI.");
    await user.click(screen.getByTestId(selectors.authoring.phaseAddButton));

    await waitFor(() => {
      expect(addPhase).toHaveBeenCalledWith("sess-1", "Authoring contract", "Add phase-native UI.");
      expect(screen.getByTestId(selectors.authoring.guidance)).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId(selectors.authoring.phaseFieldSelect), "acceptance");
    await user.type(screen.getByTestId(selectors.authoring.phaseFieldInput), "The UI can save fields.");
    await user.click(screen.getByTestId(selectors.authoring.phaseSubmitButton));

    await waitFor(() => {
      expect(submitPhaseField).toHaveBeenCalledWith("sess-1", "phase-1", "acceptance", "The UI can save fields.");
    });
  });

  it("shows degraded autofill sources honestly", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    autofill.mockResolvedValue({
      session,
      results: [
        create(AutofillResultSchema, { source: "references", filled: true, degraded: false }),
        create(AutofillResultSchema, { source: "regression_anchor", filled: false, degraded: true }),
      ],
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.autofillButton);
    await user.click(screen.getByTestId(selectors.authoring.autofillButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.authoring.autofillResults)).toBeInTheDocument();
    });
  });

  it("submits global relevant context from the wizard", async () => {
    const user = userEvent.setup();
    const contextItem = create(RelevantContextItemSchema, {
      id: "ctx-1",
      kind: RelevantContextKind.COMMAND,
      label: "Recall",
      command: "search-hub query plan-manager --type record",
    });
    startSession.mockResolvedValue({ session: sessionWithPhase, step });
    getSession.mockResolvedValue({ session: { ...sessionWithPhase, relevantContext: [contextItem] }, step });
    submitRelevantContextItem.mockResolvedValue({
      item: contextItem,
      progress: { sessionId: "sess-1" },
      violations: [],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contextLabelInput);
    await user.type(screen.getByTestId(selectors.authoring.contextLabelInput), "Recall");
    await user.type(screen.getByTestId(selectors.authoring.contextCommandInput), "search-hub query plan-manager --type record");
    await user.click(screen.getByTestId(selectors.authoring.contextSubmitButton));

    await waitFor(() => {
      expect(submitRelevantContextItem).toHaveBeenCalled();
      expect(screen.getByTestId(selectors.authoring.contextItems)).toHaveTextContent("Recall");
    });
  });

  it("submits phase-scoped relevant context from the wizard", async () => {
    const user = userEvent.setup();
    const contextItem = create(RelevantContextItemSchema, {
      id: "ctx-phase",
      kind: RelevantContextKind.DOC,
      label: "Phase docs",
      target: "docs/phase.md",
      scope: RelevantContextScope.PHASE,
      phaseId: "phase-1",
      repeatPolicy: RelevantContextRepeatPolicy.PHASE_ENTRY,
    });
    startSession.mockResolvedValue({ session: sessionWithPhase, step });
    getSession.mockResolvedValue({
      session: { ...sessionWithPhase, phaseDrafts: [{ ...phase, relevantContext: [contextItem] }] },
      step,
    });
    submitRelevantContextItem.mockResolvedValue({
      item: contextItem,
      progress: { sessionId: "sess-1" },
      violations: [],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contextPhaseToggle);
    await user.click(screen.getByTestId(selectors.authoring.contextPhaseToggle));
    await user.selectOptions(screen.getByTestId(selectors.authoring.contextKindSelect), String(RelevantContextKind.DOC));
    await user.type(screen.getByTestId(selectors.authoring.contextLabelInput), "Phase docs");
    await user.type(screen.getByTestId(selectors.authoring.contextTargetInput), "docs/phase.md");
    await user.click(screen.getByTestId(selectors.authoring.contextSubmitButton));

    await waitFor(() => {
      expect(submitRelevantContextItem).toHaveBeenCalledWith(
        "sess-1",
        "phase-1",
        expect.objectContaining({
          label: "Phase docs",
          scope: RelevantContextScope.PHASE,
          phaseId: "phase-1",
          repeatPolicy: RelevantContextRepeatPolicy.PHASE_ENTRY,
        }),
      );
      expect(screen.getByTestId(selectors.authoring.contextItems)).toHaveTextContent("Phase docs");
    });
  });

  it("discovers context candidates from concepts", async () => {
    const user = userEvent.setup();
    const sessionWithCandidate = { ...sessionWithPhase, contextCandidates: [candidate] };
    startSession.mockResolvedValue({ session: sessionWithPhase, step });
    getSession.mockResolvedValue({ session: sessionWithCandidate, step });
    discoverContextCandidates.mockResolvedValue({
      candidates: [candidate],
      progress: { sessionId: "sess-1" },
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contextConceptsInput);
    await user.clear(screen.getByTestId(selectors.authoring.contextComplexityInput));
    await user.type(screen.getByTestId(selectors.authoring.contextConceptsInput), "plan-manager context, execution resume");
    await user.type(screen.getByTestId(selectors.authoring.contextComplexityInput), "feature");
    await user.click(screen.getByTestId(selectors.authoring.contextDiscoverButton));

    await waitFor(() => {
      expect(discoverContextCandidates).toHaveBeenCalledWith(
        "sess-1",
        ["plan-manager context", "execution resume"],
        "feature",
      );
      expect(screen.getByTestId(selectors.authoring.contextCandidates)).toHaveTextContent("Recall records");
    });
  });

  it("accepts a discovered context candidate into the current phase", async () => {
    const user = userEvent.setup();
    const sessionWithCandidate = { ...sessionWithPhase, contextCandidates: [candidate] };
    const acceptedCandidate = create(ContextCandidateSchema, { ...candidate, status: "accepted" });
    startSession.mockResolvedValue({ session: sessionWithCandidate, step });
    getSession.mockResolvedValue({
      session: {
        ...sessionWithPhase,
        contextCandidates: [acceptedCandidate],
        phaseDrafts: [{ ...phase, relevantContext: [candidateItem] }],
      },
      step,
    });
    acceptContextCandidate.mockResolvedValue({
      candidate: acceptedCandidate,
      item: candidateItem,
      progress: { sessionId: "sess-1" },
      violations: [],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contextPhaseToggle);
    await user.click(screen.getByTestId(selectors.authoring.contextPhaseToggle));
    await user.click(screen.getByTestId(selectors.authoring.contextCandidateAcceptButton));

    await waitFor(() => {
      expect(acceptContextCandidate).toHaveBeenCalledWith("sess-1", "cand-1", "phase-1");
      expect(screen.getByTestId(selectors.authoring.contextItems)).toHaveTextContent("Recall records");
    });
  });

  it("rejects a discovered context candidate with a reason", async () => {
    const user = userEvent.setup();
    const sessionWithCandidate = { ...sessionWithPhase, contextCandidates: [candidate] };
    const rejectedCandidate = create(ContextCandidateSchema, {
      ...candidate,
      status: "rejected",
      rejectionReason: "duplicate",
    });
    startSession.mockResolvedValue({ session: sessionWithCandidate, step });
    getSession.mockResolvedValue({ session: { ...sessionWithPhase, contextCandidates: [rejectedCandidate] }, step });
    rejectContextCandidate.mockResolvedValue({
      candidate: rejectedCandidate,
      progress: { sessionId: "sess-1" },
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contextRejectReasonInput);
    await user.type(screen.getByTestId(selectors.authoring.contextRejectReasonInput), "duplicate");
    await user.click(screen.getByTestId(selectors.authoring.contextCandidateRejectButton));

    await waitFor(() => {
      expect(rejectContextCandidate).toHaveBeenCalledWith("sess-1", "cand-1", "duplicate");
      expect(screen.getByTestId(selectors.authoring.contextCandidates)).toHaveTextContent("duplicate");
    });
  });

  it("suggests references from search-hub and accepts a reviewed candidate", async () => {
    const user = userEvent.setup();
    const sessionWithRefCandidate = { ...sessionWithPhase, referenceCandidates: [referenceCandidate] };
    const acceptedRef = create(ReferenceCandidateSchema, { ...referenceCandidate, status: "accepted" });
    startSession.mockResolvedValue({ session: sessionWithPhase, step });
    getSession.mockResolvedValue({ session: sessionWithRefCandidate, step });
    suggestReferences.mockResolvedValue({ candidates: [referenceCandidate], progress: { sessionId: "sess-1" }, step });
    acceptReferenceCandidate.mockResolvedValue({
      candidate: acceptedRef,
      progress: { sessionId: "sess-1" },
      violations: [],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await user.click(screen.getByTestId(selectors.authoring.referenceSuggestButton));
    await waitFor(() => {
      expect(suggestReferences).toHaveBeenCalledWith("sess-1");
      expect(screen.getByTestId(selectors.authoring.referenceCandidates)).toHaveTextContent(
        "api/internal/authoring/service.go",
      );
    });

    // Re-hydrate with the accepted candidate so the post-accept render reflects it.
    getSession.mockResolvedValue({
      session: { ...sessionWithPhase, referenceCandidates: [acceptedRef] },
      step,
    });
    await user.click(screen.getByTestId(selectors.authoring.referenceCandidateAcceptButton));
    await waitFor(() => {
      expect(acceptReferenceCandidate).toHaveBeenCalledWith("sess-1", "ref-cand-1");
    });
  });

  it("renders degraded, doc, and req reference candidates with provenance", async () => {
    const user = userEvent.setup();
    const docCandidate = create(ReferenceCandidateSchema, {
      id: "ref-doc",
      reference: { id: "rd", kind: ReferenceKind.DOC, target: "docs/concepts/PLAN-MODEL.md" },
      source: "docs",
      status: "pending",
      detail: "matched the references concept",
    });
    const reqCandidate = create(ReferenceCandidateSchema, {
      id: "ref-req",
      reference: { id: "rq", kind: ReferenceKind.REQ, target: "PM-AUTHOR-002" },
      source: "requirements",
      degraded: true,
      detail: "provider degraded",
      status: "pending",
    });
    const sessionWithRefs = { ...sessionWithPhase, referenceCandidates: [docCandidate, reqCandidate] };
    startSession.mockResolvedValue({ session: sessionWithRefs, step });
    getSession.mockResolvedValue({ session: sessionWithRefs, step });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    const list = await screen.findByTestId(selectors.authoring.referenceCandidates);
    expect(list).toHaveTextContent("[DOC: docs/concepts/PLAN-MODEL.md]");
    expect(list).toHaveTextContent("[REQ: PM-AUTHOR-002]");
    expect(list).toHaveTextContent("requirements");
    expect(list).toHaveTextContent("provider degraded");
  });

  it("rejects a suggested reference candidate with a reason", async () => {
    const user = userEvent.setup();
    const sessionWithRefCandidate = { ...sessionWithPhase, referenceCandidates: [referenceCandidate] };
    const rejectedRef = create(ReferenceCandidateSchema, {
      ...referenceCandidate,
      status: "rejected",
      rejectionReason: "unrelated subsystem",
    });
    startSession.mockResolvedValue({ session: sessionWithRefCandidate, step });
    getSession.mockResolvedValue({ session: { ...sessionWithPhase, referenceCandidates: [rejectedRef] }, step });
    rejectReferenceCandidate.mockResolvedValue({ candidate: rejectedRef, progress: { sessionId: "sess-1" }, step });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.referenceRejectReasonInput);
    await user.type(screen.getByTestId(selectors.authoring.referenceRejectReasonInput), "unrelated subsystem");
    await user.click(screen.getByTestId(selectors.authoring.referenceCandidateRejectButton));

    await waitFor(() => {
      expect(rejectReferenceCandidate).toHaveBeenCalledWith("sess-1", "ref-cand-1", "unrelated subsystem");
      expect(screen.getByTestId(selectors.authoring.referenceCandidates)).toHaveTextContent("unrelated subsystem");
    });
  });

  it("removes an accepted global context item before finalize", async () => {
    const user = userEvent.setup();
    const globalItem = create(RelevantContextItemSchema, {
      id: "ctx-remove",
      kind: RelevantContextKind.NOTE,
      label: "Bad accepted note",
      instruction: "this turned out to be wrong",
      scope: RelevantContextScope.GLOBAL,
    });
    startSession.mockResolvedValue({
      session: { ...sessionWithPhase, relevantContext: [globalItem] },
      step,
    });
    // After removal the read-after-write hydration returns an empty context list.
    getSession.mockResolvedValue({ session: { ...sessionWithPhase, relevantContext: [] }, step });
    removeRelevantContextItem.mockResolvedValue({
      summary: { objectKind: "context", objectId: "ctx-remove" },
      progress: { sessionId: "sess-1" },
      violations: [],
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.contextItems);
    expect(screen.getByTestId(selectors.authoring.contextItems)).toHaveTextContent("Bad accepted note");
    await user.click(screen.getByTestId(selectors.authoring.contextRemoveButton));

    await waitFor(() => {
      expect(removeRelevantContextItem).toHaveBeenCalledWith("sess-1", "", "ctx-remove");
      expect(screen.queryByTestId(selectors.authoring.contextItems)).not.toBeInTheDocument();
    });
  });

  it("renders degraded candidates and target-only context items", async () => {
    const user = userEvent.setup();
    const targetContext = create(RelevantContextItemSchema, {
      id: "ctx-doc",
      kind: RelevantContextKind.DOC,
      label: "Plan model docs",
      target: "scenarios/plan-manager/docs/concepts/PLAN-MODEL.md",
    });
    const degradedCandidate = create(ContextCandidateSchema, {
      id: "cand-degraded",
      item: targetContext,
      concept: "plan model",
      source: "prompt-manager",
      status: "pending",
      degraded: true,
      detail: "prompt-manager unavailable",
    });
    startSession.mockResolvedValue({
      session: {
        ...sessionWithPhase,
        phaseDrafts: [completePhase],
        relevantContext: [targetContext],
        contextCandidates: [degradedCandidate],
      },
      step,
    });
    rejectContextCandidate.mockResolvedValue({
      session: {
        ...sessionWithPhase,
        phaseDrafts: [completePhase],
        relevantContext: [targetContext],
        contextCandidates: [
          create(ContextCandidateSchema, {
            ...degradedCandidate,
            status: "rejected",
            rejectionReason: "not relevant",
          }),
        ],
      },
      candidate: degradedCandidate,
      step,
    });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    const candidateList = await screen.findByTestId(selectors.authoring.contextCandidates);
    expect(candidateList).toHaveTextContent("prompt-manager unavailable");
    expect(
      screen
        .getAllByTestId(selectors.authoring.contextItems)
        .some((list) => list.textContent.includes("scenarios/plan-manager/docs/concepts/PLAN-MODEL.md")),
    ).toBe(true);

    await user.click(screen.getByTestId(selectors.authoring.contextCandidateRejectButton));
    await waitFor(() => {
      expect(rejectContextCandidate).toHaveBeenCalledWith("sess-1", "cand-degraded", "not relevant");
    });
  });

  it("finalizes into a plan and shows the success banner", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    finalize.mockResolvedValue({ plan: create(PlanSchema, { id: "plan-9", slug: "new-plan" }), step });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.finalizeButton);
    await user.click(screen.getByTestId(selectors.authoring.finalizeButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.authoring.finalizedBanner)).toBeInTheDocument();
    });
  });

  it("finalizes even when the plan id is not returned", async () => {
    const user = userEvent.setup();
    startSession.mockResolvedValue({ session, step });
    finalize.mockResolvedValue({ plan: create(PlanSchema, { slug: "new-plan" }), step });

    renderWithProviders(<AuthoringWizard />);
    await user.type(screen.getByTestId(selectors.authoring.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.authoring.startButton));

    await screen.findByTestId(selectors.authoring.finalizeButton);
    await user.click(screen.getByTestId(selectors.authoring.finalizeButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.authoring.finalizedBanner)).toBeInTheDocument();
    });
  });

  it("renders the start gate without axe violations", async () => {
    const { container } = renderWithProviders(<AuthoringWizard />);
    await screen.findByTestId(selectors.authoring.startForm);
    await expectNoA11yViolations(container);
  });
});
