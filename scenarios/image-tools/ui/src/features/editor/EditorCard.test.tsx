/**
 * EditorCard tests — focused on the editor-card surface only.
 *
 * Renders <EditorCard /> directly so failures point at editor-feature
 * behaviour, not shell composition. Follows the canonical mock-builder
 * pattern from the co-located `./mocks/ops`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeListOperationsResponse,
  makeOperationInfo,
  makeRunOpImageResult,
  makeRunOpMetadataResult,
} from "./mocks/factories";
import { makeOpsMocks } from "./mocks/ops";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

import { EditorCard } from "./EditorCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

const uploadImage = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.upload(screen.getByTestId(selectors.editor.fileInput), PNG);
};

describe("EditorCard", () => {
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

    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.error)).toBeInTheDocument();
    });
  });

  it("renders the params form and selects the first operation by default", async () => {
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.paramsForm)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.editor.empty)).toBeInTheDocument();
    // resize is first in the canonical mock list → its width field renders.
    expect(screen.getByTestId(selectors.editor.fieldInput({ name: "width" }))).toBeInTheDocument();
  });

  it("renders different params when a new operation is chosen", async () => {
    const user = userEvent.setup();
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.operationSelect)).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId(selectors.editor.operationSelect), "metadata");

    expect(
      screen.getByTestId(selectors.editor.fieldInput({ name: "strip_all" })),
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(selectors.editor.fieldInput({ name: "width" })),
    ).not.toBeInTheDocument();
  });

  it("runs the selected operation with the operation-keyed params and shows the result", async () => {
    const { runOp } = await import("../../api/ops");
    vi.mocked(runOp).mockResolvedValueOnce(makeRunOpImageResult({ width: 64, height: 32, format: "webp" }));

    const user = userEvent.setup();
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.runButton)).toBeInTheDocument();
    });

    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.editor.runButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.resultImage)).toBeInTheDocument();
    });

    expect(runOp).toHaveBeenCalledWith(
      "resize",
      PNG,
      expect.objectContaining({ fit: "fit" }),
      expect.any(Object),
    );
    expect(screen.getByTestId(selectors.editor.resultMeta).textContent).toContain("64");
    expect(screen.getByTestId(selectors.editor.downloadLink)).toHaveAttribute(
      "download",
      "result.webp",
    );
  });

  it("renders the metadata output for a metadata read", async () => {
    const { runOp } = await import("../../api/ops");
    vi.mocked(runOp).mockResolvedValueOnce(makeRunOpMetadataResult({ json: '{"format":"png"}' }));

    const user = userEvent.setup();
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.operationSelect)).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId(selectors.editor.operationSelect), "metadata");
    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.editor.runButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.metadataOutput)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.editor.metadataOutput).textContent).toContain("png");
  });

  it("exposes the overlay input only for the overlay operation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.operationSelect)).toBeInTheDocument();
    });

    expect(screen.queryByTestId(selectors.editor.overlayInput)).not.toBeInTheDocument();

    await user.selectOptions(screen.getByTestId(selectors.editor.operationSelect), "overlay");
    expect(screen.getByTestId(selectors.editor.overlayInput)).toBeInTheDocument();
  });

  it("keeps the run button disabled until an image is selected", async () => {
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.runButton)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.editor.runButton)).toBeDisabled();
  });

  it("renders the run-error state when runOp rejects", async () => {
    const { runOp } = await import("../../api/ops");
    vi.mocked(runOp).mockRejectedValueOnce(new Error("op failed"));

    const user = userEvent.setup();
    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.runButton)).toBeInTheDocument();
    });

    await uploadImage(user);
    await user.click(screen.getByTestId(selectors.editor.runButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.runError)).toBeInTheDocument();
    });
  });

  it("filters out operations without a known params spec", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockResolvedValueOnce(
      makeListOperationsResponse({
        operations: [makeOperationInfo({ name: "resize" }), makeOperationInfo({ name: "bogus" })],
      }),
    );

    renderWithProviders(<EditorCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.operationSelect)).toBeInTheDocument();
    });
    const options = screen
      .getByTestId(selectors.editor.operationSelect)
      .querySelectorAll("option");
    expect([...options].map((o) => o.getAttribute("value"))).toEqual(["resize"]);
  });
});
