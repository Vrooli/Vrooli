/**
 * EnhancePanel tests — the Enhance-mode inspector. The AI-op discovery is
 * mocked and the `UseEnhance` lifecycle is a hand-built fake, so the panel's
 * action list, upscale scale, install gate, progress, and run wiring are
 * exercised in isolation (the lifecycle itself is covered by useEnhance.test).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeAIMocks, makeSelectedModel } from "./mocks/ai";

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  return { ...actual, ...makeAIMocks() };
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

  it("renders the hardware-fit model badge with its speed note", async () => {
    const enhance = fakeEnhance({
      model: makeSelectedModel({ name: "real-esrgan", cpuCapable: false, minVramGb: 6, speedNote: "~8s" }),
    });
    renderPanel(enhance);

    const badge = await screen.findByTestId(selectors.workspace.enhance.modelBadge);
    expect(badge.textContent).toContain("real-esrgan");
    expect(badge.textContent).toContain("~8s");
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
});
