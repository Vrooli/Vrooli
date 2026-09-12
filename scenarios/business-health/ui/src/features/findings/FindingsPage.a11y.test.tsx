/**
 * FindingsPage accessibility regression — the populated, grouped findings list
 * (including a preview-fix affordance) must be axe-clean under a real locale.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/contract", () => ({
  contractClient: {
    validateScenario: vi.fn(),
    getMatrix: vi.fn(),
    getDrift: vi.fn(),
    logManualValidation: vi.fn(),
  },
}));

vi.mock("../../api/validation", () => ({
  validationClient: {
    previewFix: vi.fn(),
    applyFix: vi.fn(),
    validateScenario: vi.fn(),
  },
}));

import { FindingsPage } from "./FindingsPage";
import { contractClient } from "../../api/contract";
import {
  makeBusinessContractReport,
  makeContractFinding,
  makeValidateScenarioResponse,
} from "./mocks/factories";

describe("FindingsPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        report: makeBusinessContractReport({
          findings: [makeContractFinding({ code: "intent.ot_orphan", autofixAvailable: true })],
        }),
      }),
    );
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("has no violations for the populated findings list", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<FindingsPage />);

    await user.type(screen.getByTestId(selectors.scenarioPicker.input), "business-health");
    await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
    await screen.findByTestId(selectors.findings.list);
    await expectNoA11yViolations(container);
  });
});
