/**
 * Workspace integration tests — the unified surface that replaced the two
 * editor cards. Renders <Workspace /> with an injected op runner so the form,
 * operation select, apply, in-canvas result rendering, history, and mode
 * placeholder are exercised without the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import type { RunOpImageResult } from "../../api/ops";
import {
  makeListOperationsResponse,
  makeOperationInfo,
  makeRunOpImageResult,
} from "./mocks/factories";
import { makeOpsMocks } from "./mocks/ops";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

vi.mock("../../api/analysis", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/analysis")>();
  const { makeAnalysisMocks } = await import("./mocks/analysis");
  return { ...actual, ...makeAnalysisMocks() };
});

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  const { makeAIMocks } = await import("./mocks/ai");
  return { ...actual, ...makeAIMocks() };
});

import { Workspace } from "./Workspace";
import type { WorkspaceRunner } from "./useWorkspace";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeCreateClient } from "./mocks/ai";
import { resetWorkspaceIntent, setWorkspaceIntent } from "./workspaceIntent";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

const imageRunner =
  (overrides: Partial<RunOpImageResult> = {}): WorkspaceRunner =>
  () =>
    Promise.resolve({
      kind: "image",
      result: makeRunOpImageResult(overrides),
      outputFile: new File(["x"], "out.png", { type: "image/png" }),
    });

const uploadImage = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.upload(screen.getByTestId(selectors.workspace.fileInput), PNG);
};

describe("Workspace", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:fake"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
    resetWorkspaceIntent();
  });

  it("renders the error state when listOperations rejects", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockRejectedValueOnce(new Error("ops down"));

    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.error)).toBeInTheDocument();
    });
  });

  it("renders the params form and selects the first operation by default", async () => {
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.paramsForm)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.fieldInput({ name: "width" }))).toBeInTheDocument();
  });

  it("renders different params when a new operation is chosen", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.operationSelect)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "metadata" })));

    expect(
      screen.getByTestId(selectors.workspace.fieldInput({ name: "strip_all" })),
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(selectors.workspace.fieldInput({ name: "width" })),
    ).not.toBeInTheDocument();
  });

  it("applies the selected operation and shows the result in the canvas", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Workspace runner={imageRunner({ width: 64, height: 32, format: "webp" })} />,
    );
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });

    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.workspace.applyButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.canvas.image)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.canvas.meta).textContent).toContain("64");
    expect(screen.getByTestId(selectors.workspace.actions.download)).toHaveAttribute(
      "download",
      "result.webp",
    );
    expect(screen.getByTestId(selectors.workspace.historyStep({ index: 1 }))).toBeInTheDocument();
  });

  it("renders the metadata output for a metadata read", async () => {
    const metaRunner: WorkspaceRunner = () =>
      Promise.resolve({ kind: "metadata", json: '{"format":"png"}' });
    const user = userEvent.setup();
    renderWithProviders(<Workspace runner={metaRunner} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.operationSelect)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "metadata" })));
    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.workspace.applyButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.canvas.metadataOutput)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.canvas.metadataOutput).textContent).toContain(
      "png",
    );
  });

  it("exposes the overlay input only for the overlay operation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.operationSelect)).toBeInTheDocument();
    });

    expect(screen.queryByTestId(selectors.workspace.overlayInput)).not.toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "overlay" })));
    expect(screen.getByTestId(selectors.workspace.overlayInput)).toBeInTheDocument();
  });

  it("keeps the apply button disabled until an image is selected", async () => {
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.applyButton)).toBeDisabled();
  });

  it("renders the run-error state when the runner rejects", async () => {
    const failRunner: WorkspaceRunner = () => Promise.reject(new Error("op failed"));
    const user = userEvent.setup();
    renderWithProviders(<Workspace runner={failRunner} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });

    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.workspace.applyButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.runError)).toBeInTheDocument();
    });
  });

  it("undoes an applied step from the action bar", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace runner={imageRunner()} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });

    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.workspace.applyButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.historyStep({ index: 1 }))).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.actions.undo));
    await waitFor(() => {
      expect(
        screen.queryByTestId(selectors.workspace.historyStep({ index: 1 })),
      ).not.toBeInTheDocument();
    });
  });

  it("renders the Analyze panel when the Analyze mode is selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.modeSwitcher)).toBeInTheDocument();
    });

    // Every mode now has a real panel; Analyze surfaces the analysis ops.
    await user.click(screen.getByTestId(selectors.workspace.modeOption({ mode: "analyze" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.panel)).toBeInTheDocument();
    });
  });

  it("exposes the crop numeric fields under the Advanced disclosure (accessible fallback)", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockResolvedValueOnce(
      makeListOperationsResponse({
        operations: [makeOperationInfo({ name: "resize" }), makeOperationInfo({ name: "crop" })],
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.operationSelect)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "crop" })));

    expect(screen.getByTestId(selectors.workspace.crop.advanced)).toBeInTheDocument();
    for (const name of ["x", "y", "width", "height"]) {
      expect(screen.getByTestId(selectors.workspace.fieldInput({ name }))).toBeInTheDocument();
    }
  });

  it("filters out operations without a known params spec", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockResolvedValueOnce(
      makeListOperationsResponse({
        operations: [makeOperationInfo({ name: "resize" }), makeOperationInfo({ name: "bogus" })],
      }),
    );

    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.operationSelect)).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.workspace.opOption({ name: "resize" })),
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(selectors.workspace.opOption({ name: "bogus" })),
    ).not.toBeInTheDocument();
  });

  it("undoes and redoes an applied step via the keyboard chords", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace runner={imageRunner()} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });

    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.workspace.applyButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.historyStep({ index: 1 }))).toBeInTheDocument();
    });

    // Ctrl+Z routes through onUndo (canUndo true → handled).
    await user.keyboard("{Control>}z{/Control}");
    await waitFor(() => {
      expect(
        screen.queryByTestId(selectors.workspace.historyStep({ index: 1 })),
      ).not.toBeInTheDocument();
    });

    // Ctrl+Shift+Z routes through onRedo (canRedo true → handled).
    await user.keyboard("{Control>}{Shift>}z{/Shift}{/Control}");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.historyStep({ index: 1 }))).toBeInTheDocument();
    });
  });

  it("no-ops the undo/redo chords when there is nothing to undo or redo", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace runner={imageRunner()} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });

    // With an empty history both callbacks take the canUndo/canRedo === false
    // branch and report not-handled; the surface stays put.
    await user.keyboard("{Control>}z{/Control}");
    await user.keyboard("{Control>}y{/Control}");
    expect(
      screen.queryByTestId(selectors.workspace.historyStep({ index: 1 })),
    ).not.toBeInTheDocument();
  });

  it("seeds the crop rect from the natural image size on the canvas", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockResolvedValue(
      makeListOperationsResponse({
        operations: [makeOperationInfo({ name: "crop" }), makeOperationInfo({ name: "resize" })],
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<Workspace />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.operationSelect)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "crop" })));
    await uploadImage(user);

    // The crop overlay mounts once an image is on the canvas; firing the
    // image's load with a natural size exercises seedCrop, which (because the
    // rect is still at its default) seeds x/y/w/h into the same params the
    // accessible numeric fallback drives.
    const image = await screen.findByTestId(selectors.workspace.canvas.image);
    Object.defineProperty(image, "naturalWidth", { value: 800, configurable: true });
    Object.defineProperty(image, "naturalHeight", { value: 600, configurable: true });
    fireEvent.load(image);

    await user.click(screen.getByTestId(selectors.workspace.crop.advanced));
    const widthField = screen.getByTestId(selectors.workspace.fieldInput({ name: "width" }));
    // seedCrop pushed the full-image rect into params (non-default width).
    await waitFor(() => expect(widthField).not.toHaveValue(0));
  });

  it("sends a generated variation to Enhance, switching modes with the image", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace createClient={makeCreateClient()} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.modeSwitcher)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.modeOption({ mode: "create" })));
    await screen.findByTestId(selectors.workspace.create.panel);

    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "a serene lake");
    await user.click(screen.getByTestId(selectors.workspace.create.run));
    await screen.findByTestId(selectors.workspace.createVariation({ index: 1 }));

    // "Enhance" adopts the variation as the new base AND switches to Enhance
    // mode. Scope to the result grid so the mode-switcher's Enhance tab (same
    // accessible name) isn't matched.
    const grid = within(screen.getByTestId(selectors.workspace.create.results));
    await user.click(grid.getByRole("button", { name: /enhance/i }));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.panel)).toBeInTheDocument();
    });
  });

  it("applies a staged Home/Library handoff intent on mount (mode + AI action)", async () => {
    setWorkspaceIntent({ mode: "enhance", operation: "upscale", file: PNG });
    renderWithProviders(<Workspace />);

    // The intent opens the Enhance panel with the upscale action pre-selected.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.panel)).toBeInTheDocument();
    });
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.enhanceAction({ name: "upscale" })),
      ).toHaveAttribute("aria-checked", "true");
    });
  });

  it("applies a deterministic-op handoff intent on mount (Edit mode)", async () => {
    setWorkspaceIntent({ mode: "edit", operation: "metadata" });
    renderWithProviders(<Workspace />);

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.fieldInput({ name: "strip_all" })),
      ).toBeInTheDocument();
    });
  });
});
