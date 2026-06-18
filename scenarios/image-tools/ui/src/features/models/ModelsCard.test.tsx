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

  it("renders model metadata: size, operations, custom + NSFW badges, install state", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({
            id: "m-custom",
            operations: ["upscale", "denoise"],
            sizeMbApprox: 128,
            custom: true,
            capabilityLabels: { nsfwCapable: true, license: "MIT" },
          }),
        ],
      }),
    );

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.list)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.models.size).textContent).toContain("128");
    expect(screen.getByTestId(selectors.models.operations).textContent).toContain("denoise");
    expect(screen.getByTestId(selectors.models.customBadge)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.models.nsfwBadge)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.models.installState).textContent).toContain("Not installed");
  });

  it("installs a not-installed model via installModel and surfaces the job notice", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValue(
      makeListModelsResponse({
        models: [makeModel({ id: "m-new", install: { installed: false } })],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.installButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.models.installButton));

    await waitFor(() => {
      expect(modelsClient.installModel).toHaveBeenCalledWith({ id: "m-new" });
    });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.installNotice).textContent).toContain(
        "job-install-1",
      );
    });
  });

  it("hides the install button when the model is already installed", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [makeModel({ id: "m-have", install: { installed: true, path: "/w" } })],
      }),
    );

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.installState).textContent).toContain("Installed");
    });
    expect(screen.queryByTestId(selectors.models.installButton)).not.toBeInTheDocument();
  });

  it("surfaces the default-for badge, default-for operations, and download size", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({
            id: "m-def",
            operations: ["upscale", "denoise"],
            defaultFor: ["upscale"],
            sizeMbApprox: 64,
            install: { installed: false },
          }),
        ],
      }),
    );

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.list)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.models.defaultBadge).textContent).toContain("Default");
    expect(screen.getByTestId(selectors.models.defaultFor).textContent).toContain("upscale");
    // Not installed → show the download size, never the on-disk size.
    expect(screen.getByTestId(selectors.models.downloadSize).textContent).toContain("64");
    expect(screen.queryByTestId(selectors.models.diskSize)).not.toBeInTheDocument();
  });

  it("shows on-disk size for an installed model instead of the download size", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({
            id: "m-disk",
            install: { installed: true, path: "/w", sizeBytes: 134_217_728n }, // 128 MB
          }),
        ],
      }),
    );

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.diskSize)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.models.diskSize).textContent).toContain("128");
    expect(screen.queryByTestId(selectors.models.downloadSize)).not.toBeInTheDocument();
  });

  it("renders commercial-use status, its notes, and tone-keyed hardware chips", async () => {
    const { CommercialUse, modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValueOnce(
      makeListModelsResponse({
        models: [
          makeModel({
            id: "m-cap",
            defaultFor: [],
            capabilityLabels: {
              license: "CreativeML-OpenRAIL-M",
              commercialUse: CommercialUse.CONDITIONAL,
              commercialUseNotes: "Allowed below 1M MAU",
            },
            hardware: {
              cpuCapable: false,
              gpuRequired: true,
              minVramGb: 8,
              minRamGb: 16,
              speedNote: "~4s per image on a 4090",
            },
          }),
        ],
      }),
    );

    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.list)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.models.commercial).textContent).toContain("Conditional");
    expect(screen.getByTestId(selectors.models.commercialNotes).textContent).toContain(
      "Allowed below 1M MAU",
    );
    const hardware = screen.getByTestId(selectors.models.hardware);
    expect(hardware.textContent).toContain("Needs a GPU");
    expect(hardware.textContent).toContain("8 GB VRAM");
    expect(hardware.textContent).toContain("~4s per image on a 4090");
    // One chip per hardware-fit signal (GPU, VRAM, RAM, speed note).
    expect(screen.getAllByTestId(selectors.models.hardwareChip)).toHaveLength(4);
  });

  it("removes a model via removeModel", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listModels).mockResolvedValue(
      makeListModelsResponse({ models: [makeModel({ id: "m-del" })] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ModelsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.removeButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.models.removeButton));

    await waitFor(() => {
      expect(modelsClient.removeModel).toHaveBeenCalledWith({ id: "m-del" });
    });
  });
});
