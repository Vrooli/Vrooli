/**
 * WizardPage tests — scenario picker → session start → question interview →
 * scaffold preview → gated apply. The connect client is mocked at the module
 * boundary so tests assert component behavior against fixture proto messages,
 * not the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { setLocale } from "../../i18n";

vi.mock("../../api/wizard", () => ({
  wizardClient: {
    startSession: vi.fn(),
    submitAnswers: vi.fn(),
    previewScaffold: vi.fn(),
    applyScaffold: vi.fn(),
  },
}));

import { WizardPage } from "./WizardPage";
import { wizardClient } from "../../api/wizard";
import {
  makeAnswer,
  makeQuestion,
  makeScaffoldPreview,
  makeScaffoldResult,
  makeSessionState,
  makeCapabilityHint,
} from "./mocks/factories";

const loadScenario = async (slug = "business-health") => {
  const user = userEvent.setup();
  await user.type(screen.getByTestId(selectors.scenarioPicker.input), slug);
  await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
  return user;
};

describe("WizardPage", () => {
  beforeEach(() => {
    vi.mocked(wizardClient.startSession).mockReset();
    vi.mocked(wizardClient.submitAnswers).mockReset();
    vi.mocked(wizardClient.previewScaffold).mockReset();
    vi.mocked(wizardClient.applyScaffold).mockReset();
  });
  afterEach(async () => {
    cleanup();
    await setLocale("en");
  });

  it("starts on the choose-scenario empty state without calling the API", () => {
    renderWithProviders(<WizardPage />);
    expect(screen.getByText(strings.common.chooseScenario)).toBeInTheDocument();
    expect(wizardClient.startSession).not.toHaveBeenCalled();
  });

  it("renders the first question and progress after starting a session", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    renderWithProviders(<WizardPage />);
    await loadScenario();

    expect(await screen.findByTestId(selectors.wizard.question)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.wizard.progress)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.wizard.step({ id: "overview" }))).toBeInTheDocument();
    expect(wizardClient.startSession).toHaveBeenCalledWith(
      expect.objectContaining({ scenario: "business-health", reset: false }),
    );
  });

  it("submits a multiline answer through submitAnswers", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    vi.mocked(wizardClient.submitAnswers).mockResolvedValue(
      makeSessionState({
        answers: { overview: makeAnswer({ text: "delivers value" }) },
        remaining: [],
        complete: true,
      }),
    );
    renderWithProviders(<WizardPage />);
    const user = await loadScenario();

    await user.type(await screen.findByTestId(selectors.wizard.answerText), "delivers value");
    await user.click(screen.getByTestId(selectors.wizard.submit));

    await waitFor(() => {
      expect(wizardClient.submitAnswers).toHaveBeenCalledWith(
        expect.objectContaining({
          sessionId: "sess-1",
          answers: [expect.objectContaining({ questionId: "overview", text: "delivers value" })],
        }),
      );
    });
  });

  it("adds and removes operational-target rows in an ot_list question", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(
      makeSessionState({
        questions: [makeQuestion({ id: "ots", target: "operational_targets_p0", kind: "ot_list" })],
        remaining: ["ots"],
      }),
    );
    renderWithProviders(<WizardPage />);
    const user = await loadScenario();

    await screen.findByTestId(selectors.wizard.otList);
    expect(screen.getByTestId(selectors.wizard.otEntry({ index: 0 }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.wizard.otEntry({ index: 1 }))).not.toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.wizard.otAdd));
    const secondRow = await screen.findByTestId(selectors.wizard.otEntry({ index: 1 }));
    expect(secondRow).toBeInTheDocument();

    await user.click(within(secondRow).getByRole("button"));
    await waitFor(() => {
      expect(screen.queryByTestId(selectors.wizard.otEntry({ index: 1 }))).not.toBeInTheDocument();
    });
  });

  it("edits operational-target title and description and submits them", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(
      makeSessionState({
        questions: [makeQuestion({ id: "ots", target: "operational_targets_p0", kind: "ot_list" })],
        remaining: ["ots"],
      }),
    );
    vi.mocked(wizardClient.submitAnswers).mockResolvedValue(
      makeSessionState({ complete: true, remaining: [] }),
    );
    renderWithProviders(<WizardPage />);
    const user = await loadScenario();

    const row = await screen.findByTestId(selectors.wizard.otEntry({ index: 0 }));
    const [title, description] = within(row).getAllByRole("textbox");
    await user.type(title as HTMLElement, "Ship the thing");
    await user.type(description as HTMLElement, "delivers value");
    await user.click(screen.getByTestId(selectors.wizard.submit));

    await waitFor(() => {
      expect(wizardClient.submitAnswers).toHaveBeenCalledWith(
        expect.objectContaining({
          answers: [
            expect.objectContaining({
              targets: [expect.objectContaining({ title: "Ship the thing", description: "delivers value" })],
            }),
          ],
        }),
      );
    });
  });

  it("edits, adds, and removes entries in a list question and submits them", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(
      makeSessionState({
        questions: [
          makeQuestion({ id: "deps", target: "dependencies", kind: "list", required: false }),
        ],
        remaining: ["deps"],
      }),
    );
    vi.mocked(wizardClient.submitAnswers).mockResolvedValue(
      makeSessionState({ complete: true, remaining: [] }),
    );
    renderWithProviders(<WizardPage />);
    const user = await loadScenario();

    const question = await screen.findByTestId(selectors.wizard.question);
    // Optional questions render the optional badge.
    expect(screen.getByText(strings.wizard.optional)).toBeInTheDocument();

    // One row to start; type a value, add a second, then remove it.
    const firstInput = within(question).getAllByRole("textbox")[0];
    await user.type(firstInput as HTMLElement, "redis");
    await user.click(screen.getByRole("button", { name: strings.wizard.listAdd }));
    const removeButtons = within(question).getAllByRole("button", { name: strings.wizard.listRemove });
    await user.click(removeButtons[removeButtons.length - 1] as HTMLElement);

    await user.click(screen.getByTestId(selectors.wizard.submit));
    await waitFor(() => {
      expect(wizardClient.submitAnswers).toHaveBeenCalledWith(
        expect.objectContaining({
          answers: [expect.objectContaining({ items: ["redis"] })],
        }),
      );
    });
  });

  it("surfaces an invalid-answer reason from the session state", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(
      makeSessionState({
        answers: { overview: makeAnswer({ invalidReason: "must be at least 10 characters" }) },
      }),
    );
    renderWithProviders(<WizardPage />);
    await loadScenario();

    const invalid = await screen.findByTestId(selectors.wizard.invalid);
    expect(invalid).toHaveTextContent("must be at least 10 characters");
  });

  it("renders a scaffold preview with a file diff", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    vi.mocked(wizardClient.previewScaffold).mockResolvedValue(
      makeScaffoldPreview({ blocking: ["overview is required"] }),
    );
    renderWithProviders(<WizardPage />);
    const user = await loadScenario();

    await screen.findByTestId(selectors.wizard.question);
    await user.click(screen.getByTestId(selectors.wizard.previewButton));

    expect(await screen.findByTestId(selectors.wizard.preview)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.findings.fixDiff)).toBeInTheDocument();
    expect(screen.getByText(strings.wizard.previewBlocking)).toBeInTheDocument();
  });

  it("keeps apply disabled until the session is complete", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    renderWithProviders(<WizardPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.wizard.apply)).toBeDisabled();
  });

  it("applies the scaffold with an explicit apply flag once complete", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(
      makeSessionState({
        answers: { overview: makeAnswer() },
        remaining: [],
        complete: true,
        hints: [makeCapabilityHint()],
      }),
    );
    vi.mocked(wizardClient.applyScaffold).mockResolvedValue(
      makeScaffoldResult({ residualFindings: ["ledger drift"] }),
    );
    renderWithProviders(<WizardPage />);
    const user = await loadScenario();

    expect(screen.getByText(strings.wizard.complete)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.wizard.hints)).toBeInTheDocument();

    const apply = screen.getByTestId(selectors.wizard.apply);
    expect(apply).not.toBeDisabled();
    await user.click(apply);

    await waitFor(() => {
      expect(wizardClient.applyScaffold).toHaveBeenCalledWith(
        expect.objectContaining({ sessionId: "sess-1", apply: true }),
      );
    });
    expect(await screen.findByTestId(selectors.wizard.applyResult)).toBeInTheDocument();
    expect(screen.getByText(strings.wizard.residual)).toBeInTheDocument();
  });

  it("passes reset:true when the reset toggle is enabled", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    const user = userEvent.setup();
    renderWithProviders(<WizardPage />);

    await user.click(screen.getByTestId(selectors.wizard.resetToggle));
    await loadScenario();

    expect(wizardClient.startSession).toHaveBeenCalledWith(
      expect.objectContaining({ scenario: "business-health", reset: true }),
    );
  });

  it("renders the saved-answer section preview empty state before answers exist", async () => {
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    renderWithProviders(<WizardPage />);
    await loadScenario();
    await screen.findByTestId(selectors.wizard.sectionPreview);
    expect(screen.getByText(strings.wizard.sectionPreviewEmpty)).toBeInTheDocument();
  });

  it("surfaces an error when the session fails to start", async () => {
    vi.mocked(wizardClient.startSession).mockRejectedValue(new Error("boom"));
    renderWithProviders(<WizardPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.wizard.error)).toBeInTheDocument();
  });

  it("renders localized copy under a real locale", async () => {
    await setLocale("ja");
    vi.mocked(wizardClient.startSession).mockResolvedValue(makeSessionState());
    renderWithProviders(<WizardPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.wizard.question)).toBeInTheDocument();
  });
});
