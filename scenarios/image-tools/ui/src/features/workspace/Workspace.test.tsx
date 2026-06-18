/**
 * Workspace integration tests — the unified surface that replaced the two
 * editor cards. Renders <Workspace /> with an injected op runner so the form,
 * operation select, apply, in-canvas result rendering, history, and mode
 * placeholder are exercised without the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
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

import { Workspace } from "./Workspace";
import type { WorkspaceRunner } from "./useWorkspace";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

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
});
