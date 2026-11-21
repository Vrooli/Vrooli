/**
 * AddCustomModelForm tests — the custom/local model registration surface.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeModelsMocks } from "./mocks/models";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { AddCustomModelForm } from "./AddCustomModelForm";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("AddCustomModelForm", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("submits a custom model with parsed operations via addCustomModel", async () => {
    const { modelsClient } = await import("../../api/models");
    const user = userEvent.setup();
    renderWithProviders(<AddCustomModelForm />);

    await user.type(screen.getByTestId(selectors.models.addCustom.id), "my-local-model");
    await user.type(screen.getByTestId(selectors.models.addCustom.operations), "upscale, denoise");
    await user.type(screen.getByTestId(selectors.models.addCustom.backend), "onnx");
    await user.type(screen.getByTestId(selectors.models.addCustom.localPath), "/models/local");

    await user.click(screen.getByTestId(selectors.models.addCustom.submit));

    await waitFor(() => {
      expect(modelsClient.addCustomModel).toHaveBeenCalledWith({
        model: { id: "my-local-model", operations: ["upscale", "denoise"], backend: "onnx" },
        localPath: "/models/local",
        downloadUrl: "",
      });
    });
    expect(await screen.findByTestId(selectors.models.addCustom.success)).toBeInTheDocument();
  });

  it("keeps submit disabled until an id is entered", () => {
    renderWithProviders(<AddCustomModelForm />);
    expect(screen.getByTestId(selectors.models.addCustom.submit)).toBeDisabled();
  });

  it("surfaces an error when addCustomModel rejects", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.addCustomModel).mockRejectedValueOnce(new Error("shadows a seed model"));

    const user = userEvent.setup();
    renderWithProviders(<AddCustomModelForm />);
    await user.type(screen.getByTestId(selectors.models.addCustom.id), "dupe");
    await user.click(screen.getByTestId(selectors.models.addCustom.submit));

    expect(await screen.findByTestId(selectors.models.addCustom.error)).toBeInTheDocument();
  });
});
