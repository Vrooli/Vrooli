/**
 * CreatePanel tests — the Create-mode inspector. AI-op discovery is mocked and
 * the `UseCreate` lifecycle is a hand-built fake, so the panel's generation-op
 * list, prompt gating, run wiring, masked-op mask brush, install gate, and
 * accessibility are exercised in isolation (the lifecycle itself is covered by
 * useCreate.test).
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
});
