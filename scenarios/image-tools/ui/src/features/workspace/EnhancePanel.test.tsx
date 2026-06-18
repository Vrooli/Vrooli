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
});
