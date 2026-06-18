/**
 * CreatePanel tests — the Create-mode inspector. AI-op discovery is mocked and
 * the `UseCreate` lifecycle is a hand-built fake, so the panel's generation-op
 * list, prompt gating, run wiring, masked-op mask brush, install gate, and
 * accessibility are exercised in isolation (the lifecycle itself is covered by
 * useCreate.test).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeAIMocks, makeSelectedModel } from "./mocks/ai";

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  return { ...actual, ...makeAIMocks() };
});

import { CreatePanel } from "./CreatePanel";
import type { UseCreate } from "./useCreate";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

const fakeCreate = (overrides: Partial<UseCreate> = {}): UseCreate => ({
  phase: "idle",
  model: null,
  progress: { percent: 0, message: "", state: "unspecified" },
  tier: "",
  warnings: [],
  error: null,
  results: [],
  requestedCount: 1,
  preview: vi.fn(),
  start: vi.fn(),
  installAndRun: vi.fn(),
  cancel: vi.fn(),
  retry: vi.fn(),
  dismiss: vi.fn(),
  ...overrides,
});

const renderPanel = (create: UseCreate, input: File | null = null, inputUrl: string | null = null) =>
  renderWithProviders(
    <CreatePanel
      create={create}
      input={input}
      inputUrl={inputUrl}
      onSendToCanvas={vi.fn()}
      onSendToEnhance={vi.fn()}
    />,
  );

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("CreatePanel", () => {
  it("populates the generation-op list from discovery and shows the prompt", async () => {
    renderPanel(fakeCreate());
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.createAction({ name: "text_to_image" })),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.workspace.createAction({ name: "inpaint" })),
    ).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.prompt)).toBeInTheDocument();
  });

  it("gates run on a non-empty prompt, then starts text-to-image with the prompt", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);

    const run = await screen.findByTestId(selectors.workspace.create.run);
    expect(run).toBeDisabled();

    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "a serene lake");
    expect(run).toBeEnabled();
    await user.click(run);

    expect(create.start).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({ prompt: "a serene lake" }),
      undefined,
      undefined,
    );
  });

  it("shows the mask brush and gates a masked op until a mask exists", async () => {
    const user = userEvent.setup();
    renderPanel(fakeCreate(), PNG, "blob:img");

    await user.click(
      await screen.findByTestId(selectors.workspace.createAction({ name: "inpaint" })),
    );

    expect(screen.getByTestId(selectors.workspace.mask.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.needsMask)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();
  });

  it("renders the install gate and installs on click", async () => {
    const create = fakeCreate({
      phase: "needs-install",
      model: makeSelectedModel({ id: "sd-1.5", name: "sd-1.5", installed: false }),
    });
    const user = userEvent.setup();
    renderPanel(create);

    const gate = await screen.findByTestId(selectors.workspace.create.installGate);
    expect(gate).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.workspace.create.install));
    expect(create.installAndRun).toHaveBeenCalledTimes(1);
  });

  it("has no detectable accessibility violations", async () => {
    const { container } = renderPanel(fakeCreate());
    await screen.findByTestId(selectors.workspace.create.prompt);
    await expectNoA11yViolations(container);
  });

  it("threads size, seed, variations, negative, model-override and BYOK into the params", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);

    await user.type(
      await screen.findByTestId(selectors.workspace.create.prompt),
      "a serene lake",
    );

    // Size is offered only for text_to_image — pick the landscape preset.
    await user.click(screen.getByRole("radio", { name: /landscape/i }));
    // Bump the variation count to 3.
    await user.click(within(screen.getByTestId(selectors.workspace.create.variations)).getByText("3"));

    // Lock a seed via the number input + the lock toggle.
    await user.type(screen.getByTestId(selectors.workspace.create.seed), "42");
    await user.click(screen.getByTestId(selectors.workspace.create.seedLock));

    // Advanced disclosure: negative prompt, model override, BYOK.
    await user.type(screen.getByTestId(selectors.workspace.create.negative), "blurry");
    await user.type(screen.getByTestId(selectors.workspace.create.model), "sdxl-custom");
    await user.click(screen.getByTestId(selectors.workspace.create.byok));

    await user.click(screen.getByTestId(selectors.workspace.create.run));

    expect(create.start).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({
        prompt: "a serene lake",
        negativePrompt: "blurry",
        width: 768,
        height: 512,
        variations: 3,
        seed: 42n,
        modelOverride: "sdxl-custom",
        allowByok: true,
      }),
      undefined,
      undefined,
    );
  });

  it("randomizes and locks the seed when the dice control is used", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);

    await user.type(
      await screen.findByTestId(selectors.workspace.create.prompt),
      "a serene lake",
    );
    await user.click(screen.getByTestId(selectors.workspace.create.seedRandomize));

    // A number input is empty (value null) until randomize seeds it.
    const seed = screen.getByTestId(selectors.workspace.create.seed);
    expect(seed).not.toHaveValue(null);
    // Randomize also engages the lock so the value sticks across runs.
    expect(screen.getByTestId(selectors.workspace.create.seedLock)).toHaveAttribute(
      "aria-checked",
      "true",
    );

    await user.click(screen.getByTestId(selectors.workspace.create.run));
    expect(create.start).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({ seed: expect.any(BigInt) }),
      undefined,
      undefined,
    );
  });

  it("passes the current canvas image to an image-required op and shows the uses-current hint", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create, PNG, "blob:img");

    await user.click(
      await screen.findByTestId(selectors.workspace.createAction({ name: "image_to_image" })),
    );
    expect(screen.getByTestId(selectors.workspace.create.usesCurrent)).toBeInTheDocument();
    // image_to_image is prompt-driven, so a prompt is still required.
    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "watercolor style");
    await user.click(screen.getByTestId(selectors.workspace.create.run));

    expect(create.start).toHaveBeenCalledWith(
      "image_to_image",
      expect.objectContaining({ prompt: "watercolor style" }),
      PNG,
      undefined,
    );
  });

  it("shows the needs-image hint for an image-required op with no canvas image", async () => {
    const user = userEvent.setup();
    renderPanel(fakeCreate(), null);

    await user.click(
      await screen.findByTestId(selectors.workspace.createAction({ name: "image_to_image" })),
    );
    expect(screen.getByTestId(selectors.workspace.create.needsImage)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();
  });

  it("renders the model badge with hardware fit and speed note", async () => {
    const create = fakeCreate({
      model: makeSelectedModel({
        id: "sd-1.5",
        name: "sd-1.5",
        cpuCapable: false,
        minVramGb: 8,
        speedNote: "~20s",
      }),
    });
    renderPanel(create);
    const badge = await screen.findByTestId(selectors.workspace.create.modelBadge);
    expect(badge.textContent).toContain("sd-1.5");
    expect(badge.textContent).toContain("~20s");
  });

  it("shows live per-variation progress with a cancel control while running", async () => {
    const create = fakeCreate({
      phase: "running",
      progress: { percent: 30, message: "produced 1/3", state: "running" },
      tier: "local-cpu",
      requestedCount: 3,
    });
    const user = userEvent.setup();
    renderPanel(create);

    const progress = await screen.findByTestId(selectors.workspace.create.progress);
    // The "produced i/N" marker is localized into the produced row.
    expect(progress.textContent).toMatch(/1.*3/);
    await user.click(screen.getByTestId(selectors.workspace.create.cancel));
    expect(create.cancel).toHaveBeenCalledTimes(1);
  });

  it("shows a raw status message while running when there is no produced marker", async () => {
    const create = fakeCreate({
      phase: "running",
      progress: { percent: 10, message: "warming up the model", state: "running" },
    });
    renderPanel(create);
    expect(await screen.findByText(/warming up the model/)).toBeInTheDocument();
  });

  it("shows the installing spinner while the model installs", async () => {
    const create = fakeCreate({ phase: "installing" });
    renderPanel(create);
    expect(await screen.findByTestId(selectors.workspace.create.progress)).toBeInTheDocument();
  });

  it("lists fallback warnings, the success count, and the failure with retry", async () => {
    const user = userEvent.setup();
    const succeeded = fakeCreate({
      phase: "succeeded",
      warnings: ["fell back to CPU"],
      results: [
        {
          index: 0,
          result: {
            kind: "image",
            url: "blob:v0",
            width: 512,
            height: 512,
            format: "png",
            jobId: "gen-1",
          },
          outputFile: PNG,
        },
      ],
    });
    const { unmount } = renderPanel(succeeded);
    expect((await screen.findByTestId(selectors.workspace.create.warnings)).textContent).toContain(
      "fell back to CPU",
    );
    expect(screen.getByTestId(selectors.workspace.create.succeeded)).toBeInTheDocument();
    unmount();

    const failed = fakeCreate({ phase: "failed", error: "out of memory" });
    renderPanel(failed);
    const fail = await screen.findByTestId(selectors.workspace.create.failed);
    expect(fail.textContent).toContain("out of memory");
    await user.click(screen.getByTestId(selectors.workspace.create.retry));
    expect(failed.retry).toHaveBeenCalledTimes(1);
  });

  it("renders the discovery loading and error states", async () => {
    const { listAIOperations } = await import("../../api/ai");
    vi.mocked(listAIOperations).mockRejectedValueOnce(new Error("ai down"));
    renderPanel(fakeCreate());
    expect(await screen.findByTestId(selectors.workspace.create.error)).toBeInTheDocument();
  });
});
