import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("shiki", () => ({ codeToHtml: vi.fn().mockResolvedValue("<pre><code>highlighted</code></pre>") }));

import { renderWithProviders } from "../../test-utils";
import { DiffOp, makeDiffRow } from "./mocks/factories";
import { VersionDiffViewer } from "./VersionDiffViewer";

const rows = [
  makeDiffRow({ lineNumber: 1, text: "before", op: DiffOp.REMOVE }, { lineNumber: 1, text: "after", op: DiffOp.ADD }),
  makeDiffRow({ lineNumber: 2, text: "same", op: DiffOp.EQUAL }, { lineNumber: 2, text: "same", op: DiffOp.EQUAL }),
];

describe("VersionDiffViewer", () => {
  afterEach(() => vi.restoreAllMocks());

  it("switches modes and copies the right-side source", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    renderWithProviders(<VersionDiffViewer rows={rows} />);

    await waitFor(() => expect(screen.getAllByText("highlighted").length).toBeGreaterThan(0));
    fireEvent.click(screen.getByRole("button", { name: "versions.diff.unified" }));
    expect(screen.getByRole("button", { name: "versions.diff.unified" })).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: "versions.diff.copy" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("after\nsame"));
    expect(screen.getByRole("button", { name: "versions.diff.copied" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "versions.diff.split" }));
    expect(screen.getByRole("button", { name: "versions.diff.split" })).toHaveAttribute("aria-pressed", "true");
  });
});
