/**
 * GenerationCard tests — focused on the generation-card surface only. Renders
 * <GenerationCard /> directly so failures point at generation-feature behaviour,
 * not shell composition. Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import {
  makeImageBackendStatusResponse,
  makeImageOperationStatus,
  makeProviderStatus,
  makeProviderStatusResponse,
} from "./mocks/factories";
import { makeGenerationMocks } from "./mocks/generation";

vi.mock("../../api/generation", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/generation")>();
  return { ...actual, ...makeGenerationMocks() };
});

import { GenerationCard } from "./GenerationCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("GenerationCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("reports the unavailable summary when no provider is reachable", async () => {
    const { generationClient } = await import("../../api/generation");
    vi.mocked(generationClient.getProviderStatus).mockResolvedValueOnce(
      makeProviderStatusResponse({
        available: false,
        providers: [makeProviderStatus({ name: "ollama", available: false })],
      }),
    );

    renderWithProviders(<GenerationCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.generation.summary)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.generation.summary).textContent).toContain("No AI providers");
    expect(screen.getAllByTestId(selectors.generation.providerStatus)[0]?.textContent).toContain("Unavailable");
  });

  it("lists providers with availability when at least one is reachable", async () => {
    const { generationClient } = await import("../../api/generation");
    vi.mocked(generationClient.getProviderStatus).mockResolvedValueOnce(
      makeProviderStatusResponse({
        available: true,
        providers: [
          makeProviderStatus({ name: "ollama", available: true }),
          makeProviderStatus({ name: "openrouter", available: false }),
        ],
      }),
    );

    renderWithProviders(<GenerationCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.generation.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.generation.list);
    expect(list.textContent).toContain("ollama");
    expect(list.textContent).toContain("openrouter");
    expect(screen.getByTestId(selectors.generation.summary).textContent).toContain("At least one AI provider");
    const statuses = screen.getAllByTestId(selectors.generation.providerStatus);
    expect(statuses[0]?.textContent).toContain("Available");
    expect(statuses[1]?.textContent).toContain("Unavailable");
  });

  it("reports image-tools readiness per operation", async () => {
    const { generationClient } = await import("../../api/generation");
    vi.mocked(generationClient.getImageBackendStatus).mockResolvedValueOnce(
      makeImageBackendStatusResponse({
        available: true,
        operations: [
          makeImageOperationStatus({ operation: "generate", ready: true, modelId: "sd-1.5", tier: "local-gpu" }),
          makeImageOperationStatus({
            operation: "edit",
            ready: true,
            modelId: "openrouter-image",
            tier: "byok-cloud",
            hint: "running on a paid BYOK cloud provider",
          }),
          makeImageOperationStatus({ operation: "remove_background", ready: false, hint: "install rembg" }),
        ],
      }),
    );

    renderWithProviders(<GenerationCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.generation.imageList)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.generation.imageSummary).textContent).toContain("image-tools is reachable");
    const opStatuses = screen.getAllByTestId(selectors.generation.imageOpStatus);
    expect(opStatuses[0]?.textContent).toContain("Ready");
    expect(opStatuses[1]?.textContent).toContain("Ready");
    expect(opStatuses[2]?.textContent).toContain("Not ready");
    expect(screen.getByTestId(selectors.generation.imageList).textContent).toContain("BYOK cloud");
    expect(screen.getByTestId(selectors.generation.imageList).textContent).toContain("install rembg");
    expect(screen.getByTestId(selectors.generation.imageList).textContent).toContain("paid BYOK cloud");
  });
});
