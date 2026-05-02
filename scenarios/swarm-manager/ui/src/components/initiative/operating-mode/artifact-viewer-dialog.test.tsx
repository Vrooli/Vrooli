import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ArtifactViewerDialog } from "./artifact-viewer-dialog";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeArtifactSnapshot } from "../../../types/operating-mode";

function makeArtifact(overrides: Partial<OperatingModeArtifactSnapshot> = {}): OperatingModeArtifactSnapshot {
  return {
    path: "modes/holistic-loop/findings.md",
    contentType: "text/markdown",
    sizeBytes: 42,
    content: "# Findings\n\nObserved **lots** of things.",
    updatedAt: "2026-04-30T00:00:00Z",
    ...overrides,
  };
}

beforeEach(() => {
  // jsdom marks URL.createObjectURL as non-writable, so we install a
  // configurable property override for both methods on every test.
  Object.defineProperty(URL, "createObjectURL", {
    value: vi.fn(() => "blob:mock"),
    configurable: true,
    writable: true,
  });
  Object.defineProperty(URL, "revokeObjectURL", {
    value: vi.fn(),
    configurable: true,
    writable: true,
  });
});

describe("ArtifactViewerDialog", () => {
  it("returns null when no artifact is supplied", () => {
    render(<ArtifactViewerDialog artifact={null} isOpen onClose={() => {}} />);
    expect(screen.queryByTestId(selectors.initiativeDetails.artifactViewerDialog)).toBeNull();
  });

  it("renders markdown via renderMarkdown for text/markdown artifacts", () => {
    render(
      <ArtifactViewerDialog artifact={makeArtifact()} isOpen onClose={() => {}} />,
    );
    expect(screen.getByRole("heading", { name: /Findings/, level: 1 })).toBeInTheDocument();
    expect(screen.getByText("lots")).toBeInTheDocument();
  });

  it("falls back to a <pre> block for non-markdown content types", () => {
    render(
      <ArtifactViewerDialog
        artifact={makeArtifact({ contentType: "text/plain", content: "raw text body" })}
        isOpen
        onClose={() => {}}
      />,
    );
    const pre = screen.getByText("raw text body");
    expect(pre.tagName).toBe("PRE");
  });

  it("exposes a Download anchor with the artifact basename as filename", () => {
    render(
      <ArtifactViewerDialog
        artifact={makeArtifact({ path: "modes/holistic-loop/findings.md" })}
        isOpen
        onClose={() => {}}
      />,
    );
    const download = screen.getByTestId(selectors.initiativeDetails.artifactDownload);
    expect(download.tagName).toBe("A");
    expect(download).toHaveAttribute("download", "findings.md");
    expect(download.getAttribute("href")).toMatch(/^blob:/);
  });

  it("hides the Download anchor when the artifact has no content yet", () => {
    render(
      <ArtifactViewerDialog
        artifact={makeArtifact({ content: undefined })}
        isOpen
        onClose={() => {}}
      />,
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.artifactDownload)).toBeNull();
    expect(screen.getByText("Artifact not created yet.")).toBeInTheDocument();
  });

  it("invokes onClose when the Close button is clicked", async () => {
    const onClose = vi.fn();
    render(<ArtifactViewerDialog artifact={makeArtifact()} isOpen onClose={onClose} />);
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
