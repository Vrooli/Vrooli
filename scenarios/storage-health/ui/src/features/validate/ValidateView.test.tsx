import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ValidationStatus } from "../../api/storage";
import {
  makeFinding,
  makeFixResponse,
  makeValidateResponse,
} from "../storage/mocks/factories";

const { validateScenario, previewFix, applyFix } = vi.hoisted(() => ({
  validateScenario: vi.fn(),
  previewFix: vi.fn(),
  applyFix: vi.fn(),
}));

vi.mock("../../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/storage")>();
  return {
    ...actual,
    storageClient: { ...actual.storageClient, validateScenario, previewFix, applyFix },
  };
});

import { ValidateView } from "./ValidateView";

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ValidateView", () => {
  beforeEach(() => {
    validateScenario.mockResolvedValue(
      makeValidateResponse({
        scenario: "demo",
        status: ValidationStatus.FAILED,
        findings: [
          makeFinding({ code: "S1", severity: "SEVERITY_ERROR", autofixAvailable: true }),
          makeFinding({ code: "S2", severity: "SEVERITY_WARNING", title: "Warn" }),
        ],
        findingsBySeverity: { SEVERITY_ERROR: 1, SEVERITY_WARNING: 1 },
      }),
    );
  });

  it("shows the prompt empty state with no scenario", () => {
    renderWithProviders(<ValidateView />, { routerEntries: ["/validate"] });
    expect(screen.getByTestId(selectors.validate.prompt)).toBeInTheDocument();
    expect(validateScenario).not.toHaveBeenCalled();
  });

  it("auto-runs validation from the ?scenario= deep link", async () => {
    renderWithProviders(<ValidateView />, { routerEntries: ["/validate?scenario=demo"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validate.resultHeader)).toBeInTheDocument();
    });
    expect(validateScenario).toHaveBeenCalledWith({ scenario: "demo" });
    expect(screen.getByTestId(selectors.validate.statusPill)).toBeInTheDocument();
  });

  it("renders findings severity-sorted (error before warning)", async () => {
    renderWithProviders(<ValidateView />, { routerEntries: ["/validate?scenario=demo"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validate.findingsList)).toBeInTheDocument();
    });
    const items = screen.getAllByTestId(/^validate-finding-/);
    expect(items[0]).toHaveAttribute("data-testid", "validate-finding-0");
    expect(items[0]).toHaveTextContent("S1");
  });

  it("runs validation when a scenario is typed and submitted", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ValidateView />, { routerEntries: ["/validate"] });
    await user.type(screen.getByTestId(selectors.validate.input), "demo");
    await user.click(screen.getByTestId(selectors.validate.runButton));
    await waitFor(() => expect(validateScenario).toHaveBeenCalledWith({ scenario: "demo" }));
  });

  it("previews and applies autofix, then re-validates", async () => {
    const user = userEvent.setup();
    previewFix.mockResolvedValue(
      makeFixResponse({
        candidates: [
          { ruleId: "S1", filePath: "db.go", description: "wire seam", before: "", after: "fixed", applied: false } as never,
        ],
      }),
    );
    applyFix.mockResolvedValue(
      makeFixResponse({
        applied: true,
        candidates: [
          { ruleId: "S1", filePath: "db.go", description: "wire seam", before: "", after: "fixed", applied: true } as never,
        ],
      }),
    );
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);

    renderWithProviders(<ValidateView />, { routerEntries: ["/validate?scenario=demo"] });
    await waitFor(() => expect(screen.getByTestId(selectors.validate.previewButton)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.validate.previewButton));
    await waitFor(() => expect(screen.getByTestId(selectors.validate.candidates)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.validate.applyButton));
    await waitFor(() => expect(applyFix).toHaveBeenCalledWith({ scenario: "demo" }));
    await waitFor(() => expect(screen.getByTestId(selectors.validate.fixMessage)).toBeInTheDocument());
    // Re-validates after a non-empty apply (initial run + refetch).
    expect(validateScenario.mock.calls.length).toBeGreaterThanOrEqual(2);
    confirmSpy.mockRestore();
  });

  it("renders the clean state when there are no findings", async () => {
    validateScenario.mockResolvedValue(
      makeValidateResponse({ scenario: "ok", status: ValidationStatus.PASSED }),
    );
    renderWithProviders(<ValidateView />, { routerEntries: ["/validate?scenario=ok"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validate.clean)).toBeInTheDocument();
    });
  });

  it("renders the error state when validation throws", async () => {
    validateScenario.mockRejectedValue(new Error("boom"));
    renderWithProviders(<ValidateView />, { routerEntries: ["/validate?scenario=demo"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validate.error)).toBeInTheDocument();
    });
  });
});
