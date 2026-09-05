/**
 * ModelsCard accessibility regression tests.
 *
 * The models feature owns its query-backed loading/success/error UI plus the
 * enable/disable toggle, so a11y coverage lives here with the feature.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeListModelsResponse, makeModel } from "./mocks/factories";
import { makeModelsMocks } from "./mocks/models";
import { ModelsCard } from "./ModelsCard";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

describe("ModelsCard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state without axe violations", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(makeListModelsResponse());

    const { container } = renderWithProviders(<ModelsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.empty)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the list state without axe violations", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({ id: "m-a", name: "Real-ESRGAN", enabled: true }),
          makeModel({ id: "m-b", name: "SwinIR", enabled: false }),
        ],
      }),
    );

    const { container } = renderWithProviders(<ModelsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.list)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockRejectedValueOnce(new Error("models unavailable"));

    const { container } = renderWithProviders(<ModelsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
