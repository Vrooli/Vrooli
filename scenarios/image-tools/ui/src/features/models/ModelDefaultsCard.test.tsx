/**
 * ModelDefaultsCard tests — the per-operation default-model settings surface.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeListDefaultsResponse,
  makeListModelsResponse,
  makeModel,
  makeOpDefault,
} from "./mocks/factories";
import { makeModelsMocks } from "./mocks/models";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { ModelDefaultsCard } from "./ModelDefaultsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ModelDefaultsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists operations with their default model and source", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listDefaults).mockResolvedValueOnce(
      makeListDefaultsResponse({
        defaults: [
          makeOpDefault({ operation: "upscale", modelId: "model-1", source: "seed" }),
          makeOpDefault({ operation: "denoise", modelId: "model-2", source: "override" }),
        ],
      }),
    );
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({ id: "model-1", name: "ESRGAN", operations: ["upscale"] }),
          makeModel({ id: "model-2", name: "SCUNet", operations: ["denoise"] }),
        ],
      }),
    );

    renderWithProviders(<ModelDefaultsCard />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.models.defaults.row)).toHaveLength(2);
    });
    // The override row exposes a Clear action; the seed row does not.
    expect(screen.getAllByTestId(selectors.models.defaults.clearButton)).toHaveLength(1);
  });

  it("pins a model for an operation via setDefaultModel", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listDefaults).mockResolvedValue(
      makeListDefaultsResponse({
        defaults: [makeOpDefault({ operation: "upscale", modelId: "model-1", source: "seed" })],
      }),
    );
    vi.mocked(modelsClient.listModels).mockResolvedValue(
      makeListModelsResponse({
        models: [
          makeModel({ id: "model-1", name: "ESRGAN", operations: ["upscale"] }),
          makeModel({ id: "model-9", name: "SwinIR", operations: ["upscale"] }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ModelDefaultsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.defaults.select)).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId(selectors.models.defaults.select), "model-9");

    await waitFor(() => {
      expect(modelsClient.setDefaultModel).toHaveBeenCalledWith({
        operation: "upscale",
        modelId: "model-9",
      });
    });
  });

  it("clears a pinned default via setDefaultModel with an empty model id", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listDefaults).mockResolvedValue(
      makeListDefaultsResponse({
        defaults: [makeOpDefault({ operation: "denoise", modelId: "model-2", source: "override" })],
      }),
    );
    vi.mocked(modelsClient.listModels).mockResolvedValue(
      makeListModelsResponse({ models: [makeModel({ id: "model-2", operations: ["denoise"] })] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ModelDefaultsCard />);
    const row = await screen.findByTestId(selectors.models.defaults.row);
    await user.click(within(row).getByTestId(selectors.models.defaults.clearButton));

    await waitFor(() => {
      expect(modelsClient.setDefaultModel).toHaveBeenCalledWith({
        operation: "denoise",
        modelId: "",
      });
    });
  });
});
