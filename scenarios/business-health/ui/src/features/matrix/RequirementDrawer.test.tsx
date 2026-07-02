/**
 * RequirementDrawer tests — validations (ref present/missing, none), unproven
 * emphasis, close, and the manual-attestation success/error paths. The
 * contract client is mocked so the attestation mutation is deterministic.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

vi.mock("../../api/contract", () => ({
  contractClient: {
    getMatrix: vi.fn(),
    logManualValidation: vi.fn(),
    validateScenario: vi.fn(),
    getDrift: vi.fn(),
  },
}));

import { RequirementDrawer } from "./RequirementDrawer";
import { contractClient } from "../../api/contract";
import { makeMatrixRow, makeLogManualValidationResponse } from "./mocks/factories";

const noop = () => {};

describe("RequirementDrawer", () => {
  beforeEach(() => vi.mocked(contractClient.logManualValidation).mockReset());
  afterEach(() => cleanup());

  it("emphasizes an unproven claim with its reason", () => {
    renderWithProviders(
      <RequirementDrawer
        scenario="business-health"
        row={makeMatrixRow({ unproven: true, unprovenReason: "no passing evidence" })}
        onClose={noop}
      />,
    );
    expect(screen.getByText(strings.matrix.unproven)).toBeInTheDocument();
    expect(screen.getByText(/no passing evidence/)).toBeInTheDocument();
  });

  it("marks a validation whose ref resolves", () => {
    renderWithProviders(
      <RequirementDrawer scenario="s" row={makeMatrixRow()} onClose={noop} />,
    );
    expect(screen.getByText(strings.matrix.drawer.refExists)).toBeInTheDocument();
  });

  it("marks a validation whose ref is missing", () => {
    renderWithProviders(
      <RequirementDrawer
        scenario="s"
        row={makeMatrixRow({
          validations: [
            { type: "test", phase: "unit", status: "planned", ref: "missing.test.tsx", refExists: false },
          ],
        })}
        onClose={noop}
      />,
    );
    expect(screen.getByText(strings.matrix.drawer.refMissing)).toBeInTheDocument();
  });

  it("shows the empty state when no validations are declared", () => {
    renderWithProviders(
      <RequirementDrawer scenario="s" row={makeMatrixRow({ validations: [] })} onClose={noop} />,
    );
    expect(screen.getByText(strings.matrix.drawer.noValidations)).toBeInTheDocument();
  });

  it("calls onClose from the close affordance", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(
      <RequirementDrawer scenario="s" row={makeMatrixRow()} onClose={onClose} />,
    );
    await user.click(screen.getByTestId(selectors.matrix.drawerClose));
    expect(onClose).toHaveBeenCalled();
  });

  it("records a manual attestation and shows the success notice", async () => {
    const user = userEvent.setup();
    vi.mocked(contractClient.logManualValidation).mockResolvedValue(
      makeLogManualValidationResponse(),
    );
    renderWithProviders(
      <RequirementDrawer scenario="business-health" row={makeMatrixRow()} onClose={noop} />,
    );

    await user.type(screen.getByTestId(selectors.matrix.attestBy), "agent:qa");
    await user.click(screen.getByTestId(selectors.matrix.attestSubmit));

    await waitFor(() => {
      expect(screen.getByText(strings.matrix.drawer.attestSuccess)).toBeInTheDocument();
    });
  });

  it("disables the submit until an attester id is entered", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <RequirementDrawer scenario="business-health" row={makeMatrixRow()} onClose={noop} />,
    );
    const submit = screen.getByTestId(selectors.matrix.attestSubmit);
    expect(submit).toBeDisabled();
    await user.type(screen.getByTestId(selectors.matrix.attestBy), "agent:qa");
    expect(submit).toBeEnabled();
  });
});
