/**
 * WizardPage accessibility regression — a populated interview (active question,
 * saved-answer preview, capability hints) must be axe-clean under a real locale.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
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
import { makeAnswer, makeQuestion, makeSessionState, makeCapabilityHint } from "./mocks/factories";

describe("WizardPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(wizardClient.startSession).mockResolvedValue(
      makeSessionState({
        questions: [
          makeQuestion({ id: "overview" }),
          makeQuestion({ id: "targets", target: "operational_targets_p0", kind: "ot_list", prompt: "List the P0 outcomes." }),
        ],
        answers: { overview: makeAnswer() },
        remaining: ["targets"],
        hints: [makeCapabilityHint()],
      }),
    );
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("has no violations for a populated interview", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<WizardPage />);

    await user.type(screen.getByTestId(selectors.scenarioPicker.input), "business-health");
    await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
    await screen.findByTestId(selectors.wizard.otList);

    await expectNoA11yViolations(container);
  });
});
