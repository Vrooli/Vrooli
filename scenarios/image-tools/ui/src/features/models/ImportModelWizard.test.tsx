/**
 * ImportModelWizard tests — the guided bring-your-own-model flow.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeModelsMocks } from "./mocks/models";
import { makeInspectModelSourceResponse } from "./mocks/factories";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { ImportModelWizard } from "./ImportModelWizard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ImportModelWizard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("inspects a source, previews the proposal, then imports", async () => {
    const { modelsClient } = await import("../../api/models");
    const user = userEvent.setup();
    renderWithProviders(<ImportModelWizard />);

    await user.type(screen.getByTestId(selectors.models.import.source), "stabilityai/stable-diffusion-xl-base-1.0");
    await user.click(screen.getByTestId(selectors.models.import.inspect));

    // Preview appears with the inferred architecture + offered ops.
    const preview = await screen.findByTestId(selectors.models.import.preview);
    expect(preview).toHaveTextContent("sdxl");
    expect(preview).toHaveTextContent("text_to_image");
    expect(modelsClient.inspectModelSource).toHaveBeenCalledWith({
      source: "stabilityai/stable-diffusion-xl-base-1.0",
    });

    // Import uses the prefilled (confirmed) architecture + proposed id.
    await user.click(screen.getByTestId(selectors.models.import.submit));
    await waitFor(() => {
      expect(modelsClient.importModel).toHaveBeenCalledTimes(1);
    });
    const arg = vi.mocked(modelsClient.importModel).mock.calls[0]![0];
    expect(arg.source).toBe("stabilityai/stable-diffusion-xl-base-1.0");
    expect(arg.architecture).toBe("sdxl");
    expect(arg.id).toBe("imported-stable-diffusion-xl-base-1-0");
    expect(await screen.findByTestId(selectors.models.import.success)).toBeInTheDocument();
  });

  it("requires an explicit architecture when inference is none", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.inspectModelSource).mockResolvedValueOnce(
      makeInspectModelSourceResponse({
        architecture: { $typeName: "vrooli.image_tools.v1.models.ArchitectureInference", architecture: "none", confidence: "none", evidence: "no signal" },
        offeredOperations: [],
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<ImportModelWizard />);

    await user.type(screen.getByTestId(selectors.models.import.source), "Org/Mystery");
    await user.click(screen.getByTestId(selectors.models.import.inspect));
    await screen.findByTestId(selectors.models.import.preview);

    // Import is blocked until an architecture is chosen.
    expect(screen.getByTestId(selectors.models.import.submit)).toBeDisabled();
    await user.selectOptions(screen.getByTestId(selectors.models.import.architecture), "sd15");
    expect(screen.getByTestId(selectors.models.import.submit)).toBeEnabled();
  });
});
