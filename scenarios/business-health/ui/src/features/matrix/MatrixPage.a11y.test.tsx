/**
 * MatrixPage accessibility regression — the loaded grid and the open drawer
 * must both be axe-clean under a real locale.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/contract", () => ({
  contractClient: {
    getMatrix: vi.fn(),
    logManualValidation: vi.fn(),
    validateScenario: vi.fn(),
    getDrift: vi.fn(),
  },
}));

import { MatrixPage } from "./MatrixPage";
import { contractClient } from "../../api/contract";
import { makeGetMatrixResponse } from "./mocks/factories";

describe("MatrixPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(contractClient.getMatrix).mockResolvedValue(makeGetMatrixResponse());
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("has no violations for the loaded matrix and open drawer", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<MatrixPage />);

    await user.type(screen.getByTestId(selectors.scenarioPicker.input), "business-health");
    await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
    await screen.findByTestId(selectors.matrix.grid);
    await expectNoA11yViolations(container);

    await user.click(screen.getByTestId(selectors.matrix.drillButton({ requirementId: "BH-UX-001" })));
    await screen.findByTestId(selectors.matrix.drawer);
    await expectNoA11yViolations(container);
  });
});
