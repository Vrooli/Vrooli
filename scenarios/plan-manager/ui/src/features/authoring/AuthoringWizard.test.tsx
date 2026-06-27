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
  SectionSchema,
  StructureViolationSchema,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";
import { PlanSchema } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const startSession = vi.fn();
const addPhase = vi.fn();
const submitSection = vi.fn();
const submitPhaseField = vi.fn();
const nextPhase = vi.fn();
const validateStructure = vi.fn();
const autofill = vi.fn();
const finalize = vi.fn();

vi.mock("../../api/authoring", () => ({
  startSession: (...a: unknown[]) => startSession(...a),
  addPhase: (...a: unknown[]) => addPhase(...a),
  submitSection: (...a: unknown[]) => submitSection(...a),
  submitPhaseField: (...a: unknown[]) => submitPhaseField(...a),
  nextPhase: (...a: unknown[]) => nextPhase(...a),
  validateStructure: (...a: unknown[]) => validateStructure(...a),
  autofill: (...a: unknown[]) => autofill(...a),
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
  nextActions: [],
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
  phaseDrafts: [phase],
};

describe("AuthoringWizard", () => {
  beforeEach(async () => {
    await setLocale("en");
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
    addPhase.mockResolvedValue({
      session: sessionWithPhase,
      phase,
      violations: [],
      step,
    });
    submitPhaseField.mockResolvedValue({
      session: sessionWithPhase,
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
