/**
 * ModelPickerButton tests — the always-present control above every AI action.
 * It fires a lightweight `selectModel` query for the closed-state trigger label
 * and lazily mounts the full host-aware picker (live client) when opened, so the
 * models API is mocked to keep the test hermetic. Covers the trigger label
 * (loading → resolved → none), the GPU/CPU fit hint, opening the picker (which
 * triggers the lazy candidate load), choosing a model via onChange, and the
 * close-refetch.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeModelsMocks } from "./mocks/models";
import {
  makeCandidateModel,
  makeExplainResolutionResponse,
  makeListOperationModelsResponse,
  makeModel,
  makeResolution,
  makeSelectModelResponse,
} from "./mocks/factories";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { ModelPickerButton } from "./ModelPickerButton";

const renderButton = (props: Partial<Parameters<typeof ModelPickerButton>[0]> = {}) =>
  renderWithProviders(
    <ModelPickerButton
      operation={props.operation ?? "upscale"}
      operationLabel={props.operationLabel ?? "Upscale"}
      value={props.value ?? ""}
      onChange={props.onChange ?? vi.fn()}
    />,
  );

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ModelPickerButton", () => {
  it("renders the trigger and resolves the selected model name into its label", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.selectModel).mockResolvedValue(
      makeSelectModelResponse({ model: makeModel({ id: "m1", name: "Real-ESRGAN x4" }), gpuViable: true }),
    );
    renderButton();
    const trigger = await screen.findByTestId(selectors.models.pickerTrigger);
    await waitFor(() => expect(trigger.textContent).toContain("Real-ESRGAN x4"));
    expect(modelsClient.selectModel).toHaveBeenCalledWith({ operation: "upscale", overrideId: "" });
  });

  it("renders a 'none' label when select returns no model", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.selectModel).mockResolvedValue(makeSelectModelResponse({ model: undefined }));
    renderButton();
    // The trigger still renders even with no resolved model.
    expect(await screen.findByTestId(selectors.models.pickerTrigger)).toBeInTheDocument();
  });

  it("threads the override id into the select query", async () => {
    const { modelsClient } = await import("../../api/models");
    renderButton({ value: "override-x" });
    await screen.findByTestId(selectors.models.pickerTrigger);
    await waitFor(() =>
      expect(modelsClient.selectModel).toHaveBeenCalledWith({
        operation: "upscale",
        overrideId: "override-x",
      }),
    );
  });

  it("opens the picker on click and lazily loads the candidate menu", async () => {
    // The live picker client loads candidates via the standalone
    // `listOperationModels(operation)` helper, not `modelsClient.*`.
    const { listOperationModels } = await import("../../api/models");
    vi.mocked(listOperationModels).mockResolvedValue(
      makeListOperationModelsResponse({
        candidates: [makeCandidateModel({ model: makeModel({ id: "m1", name: "Real-ESRGAN x4" }) })],
        selectedId: "m1",
      }),
    );
    const user = userEvent.setup();
    renderButton();

    // The menu is not mounted (candidates not fetched) until opened.
    expect(listOperationModels).not.toHaveBeenCalled();

    await user.click(await screen.findByTestId(selectors.models.pickerTrigger));
    expect(screen.getByTestId(selectors.models.picker.sheet)).toBeInTheDocument();
    await waitFor(() => expect(listOperationModels).toHaveBeenCalledWith("upscale"));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.models.pickerRow({ id: "m1" }))).toBeInTheDocument(),
    );
  });

  it("surfaces a derived-op caveat under the trigger when the effective model serves the op via a workflow", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.explainResolution).mockResolvedValue(
      makeExplainResolutionResponse({
        resolution: makeResolution({
          operation: "inpaint",
          support: "derived",
          technique: "diffusers-inpaint",
          caveat: "a dedicated inpainting model blends masked edges more cleanly",
        }),
      }),
    );
    renderButton({ operation: "inpaint", operationLabel: "Inpaint" });
    const caveat = await screen.findByTestId(selectors.models.pickerTriggerCaveat);
    expect(caveat.textContent).toContain("blends masked edges more cleanly");
  });

  it("shows no derived caveat when the effective model serves the op natively", async () => {
    const { modelsClient } = await import("../../api/models");
    // Default mock resolution is native; the caveat node must be absent.
    vi.mocked(modelsClient.explainResolution).mockResolvedValue(
      makeExplainResolutionResponse({ resolution: makeResolution({ support: "native" }) }),
    );
    renderButton();
    await screen.findByTestId(selectors.models.pickerTrigger);
    await waitFor(() => expect(modelsClient.explainResolution).toHaveBeenCalled());
    expect(screen.queryByTestId(selectors.models.pickerTriggerCaveat)).toBeNull();
  });

  it("chooses a model from the menu, calling onChange", async () => {
    const { listOperationModels } = await import("../../api/models");
    vi.mocked(listOperationModels).mockResolvedValue(
      makeListOperationModelsResponse({
        candidates: [makeCandidateModel({ model: makeModel({ id: "m2", name: "SwinIR" }), readyState: "ready" })],
        selectedId: "",
      }),
    );
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderButton({ onChange });

    await user.click(await screen.findByTestId(selectors.models.pickerTrigger));
    const select = await screen.findByTestId(selectors.models.pickerSelect({ id: "m2" }));
    await user.click(select);
    expect(onChange).toHaveBeenCalledWith("m2");
  });
});
