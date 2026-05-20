import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/validation", () => {
  const validateScenario = vi.fn((scenario: string) =>
    Promise.resolve({
      scenario,
      passed: true,
      findings: [],
      summary: { errors: 0, warnings: 0, infos: 0 },
      ranAt: "2026-05-20T10:00:00.000Z",
    }),
  );
  return { validateScenario };
});

import { ValidationListPage } from "./ValidationListPage";
import { validateScenario } from "../../api/validation";

beforeEach(() => {
  window.localStorage.clear();
  vi.mocked(validateScenario).mockClear();
});

describe("ValidationListPage", () => {
  it("renders the run-validation form and empty recent state", () => {
    renderWithProviders(<ValidationListPage />);
    expect(screen.getByTestId(selectors.validation.form)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validation.scenarioInput)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validation.submit)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validation.emptyRecent)).toBeInTheDocument();
  });

  it("rejects an empty submit without calling the API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ValidationListPage />);
    const button = screen.getByTestId(selectors.validation.submit);
    expect(button).toBeDisabled();
    await user.type(screen.getByTestId(selectors.validation.scenarioInput), "   ");
    expect(button).toBeDisabled();
    expect(validateScenario).not.toHaveBeenCalled();
  });

  it("rejects an invalid scenario name and shows an error", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ValidationListPage />);
    await user.type(screen.getByTestId(selectors.validation.scenarioInput), "Bad Name!");
    await user.click(screen.getByTestId(selectors.validation.submit));
    expect(validateScenario).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("submits a valid name and records a recent run", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ValidationListPage />);
    await user.type(screen.getByTestId(selectors.validation.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.validation.submit));
    await waitFor(() => expect(validateScenario).toHaveBeenCalledWith("ui-health"));
    await waitFor(() =>
      expect(JSON.parse(window.localStorage.getItem("ui-health.validation.recent.v1") ?? "[]")).toHaveLength(1),
    );
  });
});
