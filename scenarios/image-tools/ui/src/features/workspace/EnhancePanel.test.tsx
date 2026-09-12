/**
 * EnhancePanel tests — the Enhance-mode inspector. The AI-op discovery is
 * mocked and the `UseEnhance` lifecycle is a hand-built fake, so the panel's
 * action list, upscale scale, install gate, progress, and run wiring are
 * exercised in isolation (the lifecycle itself is covered by useEnhance.test).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeAIMocks, makeSelectedModel } from "./mocks/ai";
import { makeModelsMocks } from "../models/mocks/models";
import { makeModel } from "../models/mocks/factories";

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  return { ...actual, ...makeAIMocks() };
});

// The panel renders <ModelPickerButton/>, which fires a real `selectModel`
// query for its trigger label. Mock the models API so the panel renders
// hermetically (no network) — the picker itself is covered by its own tests.
vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { EnhancePanel } from "./EnhancePanel";
import type { UseEnhance } from "./useEnhance";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

const fakeEnhance = (overrides: Partial<UseEnhance> = {}): UseEnhance => ({
  phase: "idle",
  model: null,
  progress: { percent: 0, message: "", state: "unspecified" },
  tier: "",
  warnings: [],
  error: null,
  preview: vi.fn(),
  start: vi.fn(),
  installAndRun: vi.fn(),
  cancel: vi.fn(),
  retry: vi.fn(),
  dismiss: vi.fn(),
  ...overrides,
});

const renderPanel = (enhance: UseEnhance, input: File | null = PNG) =>
  renderWithProviders(
    <EnhancePanel enhance={enhance} input={input} inputWidth={100} inputHeight={50} />,
  );

describe("EnhancePanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists only the enhancement ops and runs the selected one with its input", async () => {
    const enhance = fakeEnhance();
    const user = userEvent.setup();
    renderPanel(enhance);

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.enhanceAction({ name: "background_removal" })),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.workspace.enhanceAction({ name: "upscale" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.workspace.enhanceAction({ name: "denoise" })),
    ).toBeInTheDocument();
    // generation ops are filtered out of the enhance surface.
    expect(
      screen.queryByTestId(selectors.workspace.enhanceAction({ name: "text_to_image" })),
    ).not.toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.workspace.enhance.run));
    expect(enhance.start).toHaveBeenCalledWith("background_removal", {}, PNG);
  });

  it("offers the upscale factor and passes it as a typed param", async () => {
    const enhance = fakeEnhance();
    const user = userEvent.setup();
    renderPanel(enhance);

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.enhanceAction({ name: "upscale" })),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.enhanceAction({ name: "upscale" })));
    expect(screen.getByTestId(selectors.workspace.enhance.scale)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.workspace.enhance.run));
    expect(enhance.start).toHaveBeenCalledWith("upscale", { scale: 2 }, PNG);
  });

  it("renders the install gate and installs the model on confirm", async () => {
    const enhance = fakeEnhance({
      phase: "needs-install",
      model: makeSelectedModel({ installed: false }),
    });
    const user = userEvent.setup();
    renderPanel(enhance);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.installGate)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.workspace.enhance.install));
    expect(enhance.installAndRun).toHaveBeenCalled();
  });

  it("shows live progress with a cancel control while running", async () => {
    const enhance = fakeEnhance({
      phase: "running",
      progress: { percent: 42, message: "denoising", state: "running" },
      tier: "local-cpu",
    });
    const user = userEvent.setup();
    renderPanel(enhance);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.progress)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.workspace.enhance.cancel));
    expect(enhance.cancel).toHaveBeenCalled();
  });

  it("disables the run action when no image is loaded", async () => {
    const enhance = fakeEnhance();
    renderPanel(enhance, null);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.run)).toBeDisabled();
    });
  });

  it("renders the idle surface without axe violations", async () => {
    const enhance = fakeEnhance();
    const { container } = renderPanel(enhance);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.run)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });

  it("shows the upscale target resolution and a memory warning for a huge result", async () => {
    const enhance = fakeEnhance();
    const user = userEvent.setup();
    // 6000×6000 base × 4 → 576 MP, far over the 24 MP warning ceiling.
    renderWithProviders(
      <EnhancePanel enhance={enhance} input={PNG} inputWidth={6000} inputHeight={6000} />,
    );

    await user.click(
      await screen.findByTestId(selectors.workspace.enhanceAction({ name: "upscale" })),
    );
    // Default scale is 2×; bump to 4× so the projected result trips the warning.
    await user.click(screen.getByRole("radio", { name: "4×" }));

    // Target resolution preview (from → to) renders for a sized upscale input…
    expect(screen.getByText(/24000×24000/)).toBeInTheDocument();
    // …and the over-budget projection surfaces the memory warning.
    expect(screen.getByText(/Large result/i)).toBeInTheDocument();
  });

  it("renders the host-aware model picker trigger inside the model badge", async () => {
    const { modelsClient } = await import("../../api/models");
    const { makeSelectModelResponse } = await import("../models/mocks/factories");
    vi.mocked(modelsClient.selectModel).mockResolvedValue(
      makeSelectModelResponse({ model: makeModel({ id: "real-esrgan", name: "real-esrgan" }) }),
    );
    renderPanel(fakeEnhance());

    const badge = await screen.findByTestId(selectors.workspace.enhance.modelBadge);
    // The static "model detected" badge was replaced by the picker trigger.
    const trigger = within(badge).getByTestId(selectors.models.pickerTrigger);
    expect(trigger).toBeInTheDocument();
    await waitFor(() => expect(trigger.textContent).toContain("real-esrgan"));
  });

  it("lists fallback-tier warnings when the run reports them", async () => {
    const enhance = fakeEnhance({ warnings: ["GPU unavailable, fell back to CPU"] });
    renderPanel(enhance);

    const warnings = await screen.findByTestId(selectors.workspace.enhance.warnings);
    expect(warnings.textContent).toContain("GPU unavailable");
  });

  it("renders the success state when the job succeeds", async () => {
    const enhance = fakeEnhance({ phase: "succeeded" });
    renderPanel(enhance);
    expect(await screen.findByTestId(selectors.workspace.enhance.succeeded)).toBeInTheDocument();
  });

  it("shows the failure with its error and retries on click", async () => {
    const enhance = fakeEnhance({ phase: "failed", error: "model crashed" });
    const user = userEvent.setup();
    renderPanel(enhance);

    const failed = await screen.findByTestId(selectors.workspace.enhance.failed);
    expect(failed.textContent).toContain("model crashed");
    await user.click(screen.getByTestId(selectors.workspace.enhance.retry));
    expect(enhance.retry).toHaveBeenCalledTimes(1);
  });

  it("falls back to the generic failure copy when no error message is present", async () => {
    const enhance = fakeEnhance({ phase: "failed", error: null });
    renderPanel(enhance);
    const failed = await screen.findByTestId(selectors.workspace.enhance.failed);
    expect(failed).toHaveTextContent(/\S/);
  });

  it("renders the discovery error state when op discovery rejects", async () => {
    const { listAIOperations } = await import("../../api/ai");
    vi.mocked(listAIOperations).mockRejectedValueOnce(new Error("ai down"));
    renderPanel(fakeEnhance());
    expect(await screen.findByTestId(selectors.workspace.enhance.error)).toBeInTheDocument();
  });

  it("offers the realism knob + face-aware toggle and passes them as typed params", async () => {
    const enhance = fakeEnhance();
    const user = userEvent.setup();
    renderPanel(enhance);

    await user.click(
      await screen.findByTestId(selectors.workspace.enhanceAction({ name: "naturalize" })),
    );
    const realism = screen.getByTestId(selectors.workspace.enhance.realism);
    fireEvent.change(realism, { target: { value: "0.8" } });
    await user.click(screen.getByTestId(selectors.workspace.enhance.faceAware));

    await user.click(screen.getByTestId(selectors.workspace.enhance.run));
    expect(enhance.start).toHaveBeenCalledWith(
      "naturalize",
      { realism: 0.8, faceAware: true },
      PNG,
    );
  });

  it("suggests Naturalize after an over-smoothing op succeeds, and selecting it shows the knob", async () => {
    const enhance = fakeEnhance();
    const user = userEvent.setup();
    const { rerender } = renderPanel(enhance);

    // Run upscale (records it as the last op).
    await user.click(
      await screen.findByTestId(selectors.workspace.enhanceAction({ name: "upscale" })),
    );
    await user.click(screen.getByTestId(selectors.workspace.enhance.run));
    expect(enhance.start).toHaveBeenCalledWith("upscale", { scale: 2 }, PNG);

    // The job succeeds → the panel nudges toward Naturalize.
    const succeeded = fakeEnhance({ phase: "succeeded" });
    rerender(<EnhancePanel enhance={succeeded} input={PNG} inputWidth={100} inputHeight={50} />);
    const suggest = await screen.findByTestId(selectors.workspace.enhance.suggest);
    expect(suggest).toBeInTheDocument();

    // Accepting the suggestion switches to naturalize and reveals its knob.
    await user.click(screen.getByText(/Naturalize this result/i));
    expect(succeeded.dismiss).toHaveBeenCalled();
    expect(await screen.findByTestId(selectors.workspace.enhance.realism)).toBeInTheDocument();
  });

  it("does not suggest Naturalize after a non-smoothing op (background removal) succeeds", async () => {
    const enhance = fakeEnhance();
    const user = userEvent.setup();
    const { rerender } = renderPanel(enhance);

    // background_removal is the default-selected op; run it.
    await user.click(await screen.findByTestId(selectors.workspace.enhance.run));
    expect(enhance.start).toHaveBeenCalledWith("background_removal", {}, PNG);

    const succeeded = fakeEnhance({ phase: "succeeded" });
    rerender(<EnhancePanel enhance={succeeded} input={PNG} inputWidth={100} inputHeight={50} />);
    await screen.findByTestId(selectors.workspace.enhance.succeeded);
    expect(screen.queryByTestId(selectors.workspace.enhance.suggest)).not.toBeInTheDocument();
  });
});
