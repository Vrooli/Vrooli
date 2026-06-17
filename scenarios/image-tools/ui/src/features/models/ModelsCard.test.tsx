/**
 * ModelsCard tests — focused on the models-card surface only.
 *
 * Renders <ModelsCard /> directly so failures point at models-feature
 * behaviour, not shell composition. Follows the canonical mock-builder
 * pattern from the co-located `./mocks/models`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeListModelsResponse, makeModel } from "./mocks/factories";
import { makeModelsMocks } from "./mocks/models";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { ModelsCard } from "./ModelsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ModelsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listModels resolves with no models", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(makeListModelsResponse());

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.models.list)).not.toBeInTheDocument();
  });

  it("renders the list with name, tier, backend, and enabled state", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({ id: "m-a", name: "Real-ESRGAN", tier: "default", backend: "onnx", enabled: true }),
          makeModel({ id: "m-b", name: "SwinIR", tier: "quality", backend: "torch", enabled: false }),
        ],
      }),
    );

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.models.list);
    expect(list.textContent).toContain("Real-ESRGAN");
    expect(list.textContent).toContain("SwinIR");
    expect(screen.getAllByTestId(selectors.models.tier)[0]?.textContent).toContain("default");
    expect(screen.getAllByTestId(selectors.models.backend)[1]?.textContent).toContain("torch");
  });

  it("renders the error state when listModels rejects", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockRejectedValueOnce(new Error("models unavailable"));

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.error)).toBeInTheDocument();
    });
  });

  it("toggles an enabled model off via setModelEnabled", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValue(
      makeListModelsResponse({ models: [makeModel({ id: "m-on", enabled: true })] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.toggleButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.models.toggleButton));

    await waitFor(() => {
      expect(modelsClient.setModelEnabled).toHaveBeenCalledWith({ id: "m-on", enabled: false });
    });
  });

  it("toggles a disabled model on via setModelEnabled", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValue(
      makeListModelsResponse({ models: [makeModel({ id: "m-off", enabled: false })] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.toggleButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.models.toggleButton));

    await waitFor(() => {
      expect(modelsClient.setModelEnabled).toHaveBeenCalledWith({ id: "m-off", enabled: true });
    });
  });
});
