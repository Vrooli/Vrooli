import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders, makeApiMocks } from "../test-utils";
import { ValidationPage } from "./ValidationPage";

vi.mock("../api/validation", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/validation")>();
  return { ...actual, ...makeApiMocks() };
});

describe("ValidationPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders validation evidence returned by the provider", async () => {
    const { validateScenario } = await import("../api/validation");
    const user = userEvent.setup();

    renderWithProviders(<ValidationPage />);
    await user.click(screen.getByRole("button", { name: strings.pages.validation.validate }));

    await waitFor(() => {
      expect(validateScenario).toHaveBeenCalledWith({
        scenario: "api-health",
        path: "",
        includeExecution: false,
      });
    });
    expect(screen.getByTestId(selectors.validation.summary)).toHaveTextContent("failed");
    expect(screen.getByTestId(selectors.validation.summary)).toHaveTextContent("resolved");
    expect(screen.getByTestId(selectors.validation.probe)).toHaveTextContent("503");
    expect(screen.getByTestId(selectors.validation.capabilities)).toHaveTextContent("API lifecycle");
    expect(screen.getByTestId(selectors.validation.findings)).toHaveTextContent(
      "APIH_LIFE_MISSING_HEALTH_METADATA",
    );
  });

  it("passes path and execution mode through to provider validation", async () => {
    const { validateScenario } = await import("../api/validation");
    const user = userEvent.setup();

    renderWithProviders(<ValidationPage />);
    await user.clear(screen.getByTestId(selectors.validation.scenarioInput));
    await user.type(screen.getByTestId(selectors.validation.scenarioInput), "target-api");
    await user.type(screen.getByTestId(selectors.validation.pathInput), "/tmp/target-api");
    await user.click(screen.getByTestId(selectors.validation.executionToggle));
    await user.click(screen.getByRole("button", { name: strings.pages.validation.validate }));

    await waitFor(() => {
      expect(validateScenario).toHaveBeenCalledWith({
        scenario: "target-api",
        path: "/tmp/target-api",
        includeExecution: true,
      });
    });
  });

  it("renders deterministic fix preview candidates", async () => {
    const { previewFix } = await import("../api/validation");
    const user = userEvent.setup();

    renderWithProviders(<ValidationPage />);
    await user.click(screen.getByTestId(selectors.validation.fixPreviewButton));

    await waitFor(() => {
      expect(previewFix).toHaveBeenCalledWith({
        scenario: "api-health",
        path: "",
      });
    });
    expect(screen.getByTestId(selectors.validation.fixPreview)).toHaveTextContent(
      "APIH_LIFE_MISSING_HEALTH_METADATA",
    );
    expect(screen.getByTestId(selectors.validation.fixPreview)).toHaveTextContent(
      ".vrooli/service.json",
    );
  });
});
