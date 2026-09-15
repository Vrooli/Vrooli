/**
 * MatrixPage tests — scenario picker → matrix render → drill-in → attestation.
 * The connect client is mocked at the module boundary so tests assert
 * component behavior against fixture proto messages, not the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
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
import {
  makeGetMatrixResponse,
  makeMatrixRow,
  makeLogManualValidationResponse,
} from "./mocks/factories";

const loadScenario = async (slug = "business-health") => {
  const user = userEvent.setup();
  await user.type(screen.getByTestId(selectors.scenarioPicker.input), slug);
  await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
  return user;
};

describe("MatrixPage", () => {
  beforeEach(() => {
    vi.mocked(contractClient.getMatrix).mockReset();
    vi.mocked(contractClient.logManualValidation).mockReset();
  });
  afterEach(async () => {
    cleanup();
    await setLocale("en");
  });

  it("starts on the choose-scenario empty state without calling the API", () => {
    renderWithProviders(<MatrixPage />);
    expect(screen.getByText(strings.common.chooseScenario)).toBeInTheDocument();
    expect(contractClient.getMatrix).not.toHaveBeenCalled();
  });

  it("shows a loading state while the matrix query is in flight", async () => {
    vi.mocked(contractClient.getMatrix).mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<MatrixPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.matrix.loading)).toBeInTheDocument();
  });

  it("renders the registry summary and grouped rows on success", async () => {
    vi.mocked(contractClient.getMatrix).mockResolvedValue(makeGetMatrixResponse());
    renderWithProviders(<MatrixPage />);
    await loadScenario();

    expect(await screen.findByTestId(selectors.matrix.grid)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.matrix.registrySummary)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.matrix.requirementRow({ requirementId: "BH-UX-001" })),
    ).toBeInTheDocument();
  });

  it("emphasizes unproven claims", async () => {
    vi.mocked(contractClient.getMatrix).mockResolvedValue(
      makeGetMatrixResponse({
        matrix: [makeMatrixRow({ unproven: true, unprovenReason: "no passing evidence" })],
      }),
    );
    renderWithProviders(<MatrixPage />);
    await loadScenario();

    const row = await screen.findByTestId(
      selectors.matrix.requirementRow({ requirementId: "BH-UX-001" }),
    );
    expect(within(row).getByText(strings.matrix.unproven)).toBeInTheDocument();
  });

  it("surfaces a degraded-evidence banner when present", async () => {
    vi.mocked(contractClient.getMatrix).mockResolvedValue(
      makeGetMatrixResponse({ degradedReason: "snapshot missing" }),
    );
    renderWithProviders(<MatrixPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.matrix.degradedBanner)).toBeInTheDocument();
  });

  it("flags orphan targets and unlinked requirements", async () => {
    vi.mocked(contractClient.getMatrix).mockResolvedValue(
      makeGetMatrixResponse({
        matrix: [
          makeMatrixRow({ otId: "OT-P0-009", otTitle: "Lonely target", otChecked: false, requirementId: "" }),
          makeMatrixRow({ otId: "OT-P0-010", requirementId: "R-PLANNED", requirementStatus: "planned" }),
          makeMatrixRow({
            otId: "",
            requirementId: "R-ORPHAN",
            requirementTitle: "Unlinked requirement",
            requirementStatus: "failing",
          }),
        ],
      }),
    );
    renderWithProviders(<MatrixPage />);
    await loadScenario();

    expect(await screen.findByText(strings.matrix.orphanTarget)).toBeInTheDocument();
    expect(screen.getByText(strings.matrix.orphanRequirement)).toBeInTheDocument();
    expect(screen.getByText(strings.matrix.unchecked)).toBeInTheDocument();
  });

  it("renders an error alert when the query fails", async () => {
    vi.mocked(contractClient.getMatrix).mockRejectedValue(new Error("boom"));
    renderWithProviders(<MatrixPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.matrix.error)).toBeInTheDocument();
  });

  it("opens the drawer on drill-in and logs a manual attestation", async () => {
    vi.mocked(contractClient.getMatrix).mockResolvedValue(makeGetMatrixResponse());
    vi.mocked(contractClient.logManualValidation).mockResolvedValue(
      makeLogManualValidationResponse(),
    );
    renderWithProviders(<MatrixPage />);
    const user = await loadScenario();

    await user.click(
      await screen.findByTestId(selectors.matrix.drillButton({ requirementId: "BH-UX-001" })),
    );
    expect(await screen.findByTestId(selectors.matrix.drawer)).toBeInTheDocument();

    await user.type(screen.getByTestId(selectors.matrix.attestBy), "agent:reviewer");
    await user.click(screen.getByTestId(selectors.matrix.attestSubmit));

    await waitFor(() => {
      expect(contractClient.logManualValidation).toHaveBeenCalledWith(
        expect.objectContaining({ scenario: "business-health", requirementId: "BH-UX-001", attestedBy: "agent:reviewer" }),
      );
    });
  });

  it("renders localized copy under a real locale", async () => {
    await setLocale("ja");
    vi.mocked(contractClient.getMatrix).mockResolvedValue(makeGetMatrixResponse());
    renderWithProviders(<MatrixPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.matrix.grid)).toBeInTheDocument();
  });
});
