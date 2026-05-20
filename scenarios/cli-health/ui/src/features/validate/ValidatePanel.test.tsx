import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { Severity } from "@vrooli/proto-types/cli-health/v1/validation/validation_pb";

vi.mock("../../api/clients", () => ({
  searchClient: { search: vi.fn(), status: vi.fn() },
  validationClient: { validateScenario: vi.fn() },
  reindexClient: { reindex: vi.fn() },
}));

import { validationClient } from "../../api/clients";
import { ValidatePanel } from "./ValidatePanel";

describe("ValidatePanel", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a passing report", async () => {
    vi.mocked(validationClient.validateScenario).mockResolvedValue({
      scenario: "development-toolchain-validator",
      passed: true,
      findings: [],
      summary: { errors: 0, warnings: 0, infos: 0 },
    } as never);

    renderWithProviders(<ValidatePanel />);

    fireEvent.change(screen.getByTestId(selectors.validate.input), {
      target: { value: "development-toolchain-validator" },
    });
    fireEvent.click(screen.getByTestId(selectors.validate.submit));

    expect(await screen.findByTestId(selectors.validate.passed)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validate.empty)).toBeInTheDocument();
    expect(validationClient.validateScenario).toHaveBeenCalledWith({
      scenario: "development-toolchain-validator",
    });
  });

  it("renders findings with severity badges when validation fails", async () => {
    vi.mocked(validationClient.validateScenario).mockResolvedValue({
      scenario: "broken",
      passed: false,
      findings: [
        {
          severity: Severity.ERROR,
          code: "proto.orphan_method",
          location: "RecordingsService.GetCookies",
          message: "method not bound",
          suggestion: "add a command or mark omitted",
        },
        {
          severity: Severity.WARNING,
          code: "omission.orphan",
          location: "",
          message: "stale omission entry",
          suggestion: "",
        },
      ],
      summary: { errors: 1, warnings: 1, infos: 0 },
    } as never);

    renderWithProviders(<ValidatePanel />);

    fireEvent.change(screen.getByTestId(selectors.validate.input), {
      target: { value: "broken" },
    });
    fireEvent.click(screen.getByTestId(selectors.validate.submit));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.validate.failed)).toBeInTheDocument(),
    );
    const findings = screen.getAllByTestId(selectors.validate.finding);
    expect(findings).toHaveLength(2);
  });

  it("renders an error when the client rejects", async () => {
    vi.mocked(validationClient.validateScenario).mockRejectedValue(new Error("nope"));

    renderWithProviders(<ValidatePanel />);
    fireEvent.change(screen.getByTestId(selectors.validate.input), {
      target: { value: "x" },
    });
    fireEvent.click(screen.getByTestId(selectors.validate.submit));

    expect(await screen.findByTestId(selectors.validate.error)).toBeInTheDocument();
  });
});
