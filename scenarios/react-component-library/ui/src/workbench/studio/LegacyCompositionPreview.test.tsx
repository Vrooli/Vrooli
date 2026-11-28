import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { strings } from "../../consts/strings.generated";
import { renderWithProviders } from "../../test-utils";
import { renderLegacyComposition } from "../workspaces/legacyCompositionPreview";
import { LegacyCompositionPreview } from "./LegacyCompositionPreview";

vi.mock("../workspaces/legacyCompositionPreview", () => ({
  renderLegacyComposition: vi.fn(),
}));

const renderComposition = vi.mocked(renderLegacyComposition);

describe("LegacyCompositionPreview", () => {
  beforeEach(() => {
    renderComposition.mockReset();
  });

  it("shows a loading state until the read-only renderer resolves", async () => {
    let resolve!: (html: string) => void;
    renderComposition.mockReturnValue(new Promise((done) => (resolve = done)));

    renderWithProviders(<LegacyCompositionPreview argsJson='{"page":"dashboard"}' />);

    expect(screen.getByRole("status")).toHaveTextContent(strings.workbench.studio.legacyPreviewLoading);
    resolve("<main>Legacy page</main>");
    await waitFor(() =>
      expect(screen.getByTitle(strings.workbench.studio.legacyPreviewTitle)).toHaveAttribute(
        "srcdoc",
        "<main>Legacy page</main>",
      ),
    );
  });

  it("renders resolved HTML in a sandboxed, no-referrer frame", async () => {
    renderComposition.mockResolvedValue("<main>Read-only source</main>");

    renderWithProviders(<LegacyCompositionPreview argsJson='{"page":"conversations"}' />);

    const frame = await screen.findByTitle(strings.workbench.studio.legacyPreviewTitle);
    expect(frame).toHaveAttribute("sandbox", "allow-scripts");
    expect(frame).toHaveAttribute("referrerpolicy", "no-referrer");
    expect(frame).toHaveAttribute("srcdoc", "<main>Read-only source</main>");
    expect(renderComposition).toHaveBeenCalledWith('{"page":"conversations"}');
  });

  it("reports renderer errors and falls back for non-Error failures", async () => {
    renderComposition.mockRejectedValueOnce(new Error("preview unavailable"));
    const view = renderWithProviders(<LegacyCompositionPreview argsJson="first" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("preview unavailable");

    renderComposition.mockRejectedValueOnce("unexpected rejection");
    view.rerender(<LegacyCompositionPreview argsJson="second" />);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      strings.workbench.studio.legacyPreviewFailed,
    );
  });

  it("ignores stale results after the source arguments change", async () => {
    let resolveOld!: (html: string) => void;
    let resolveNew!: (html: string) => void;
    renderComposition
      .mockReturnValueOnce(new Promise((done) => (resolveOld = done)))
      .mockReturnValueOnce(new Promise((done) => (resolveNew = done)));
    const view = renderWithProviders(<LegacyCompositionPreview argsJson="old" />);

    view.rerender(<LegacyCompositionPreview argsJson="new" />);
    resolveOld("<main>Stale</main>");
    expect(screen.getByRole("status")).toBeInTheDocument();
    resolveNew("<main>Current</main>");

    expect(await screen.findByTitle(strings.workbench.studio.legacyPreviewTitle)).toHaveAttribute(
      "srcdoc",
      "<main>Current</main>",
    );
  });
});
