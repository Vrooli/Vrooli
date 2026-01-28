/**
 * FileTree Component Tests
 *
 * [REQ:REQ-P0-004] Tests for file tree component rendering and interactions
 */

import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { FileTree } from "./file-tree";
import type { IdeaFile } from "../../types";

describe("FileTree", () => {
  // [REQ:REQ-P0-004] Test empty file tree displays empty state
  it("displays empty state when no files", () => {
    render(<FileTree files={[]} />);
    expect(screen.getByText("No files yet")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-004] Test file tree renders files
  it("renders file list correctly", () => {
    const files: IdeaFile[] = [
      { name: "spec.json", path: "spec.json", type: "file", size: 256 },
      { name: "notes.md", path: "notes.md", type: "file", size: 1024 },
    ];

    render(<FileTree files={files} />);

    expect(screen.getByText("spec.json")).toBeInTheDocument();
    expect(screen.getByText("notes.md")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-004] Test file tree renders directories with expand/collapse
  it("renders directories that can be expanded", () => {
    const files: IdeaFile[] = [
      {
        name: "research",
        path: "research",
        type: "directory",
        children: [
          { name: "findings.md", path: "research/findings.md", type: "file", size: 512 },
        ],
      },
    ];

    render(<FileTree files={files} />);

    // Directory should be visible
    expect(screen.getByText("research")).toBeInTheDocument();

    // Children should not be visible initially
    expect(screen.queryByText("findings.md")).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(screen.getByText("research"));

    // Children should now be visible
    expect(screen.getByText("findings.md")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-004] Test file selection callback
  it("calls onFileSelect when file is clicked", () => {
    const onFileSelect = vi.fn();
    const files: IdeaFile[] = [
      { name: "spec.json", path: "spec.json", type: "file", size: 256 },
    ];

    render(<FileTree files={files} onFileSelect={onFileSelect} />);

    fireEvent.click(screen.getByText("spec.json"));

    expect(onFileSelect).toHaveBeenCalledWith(files[0]);
  });

  // [REQ:REQ-P0-004] Test file size formatting
  it("displays file size in human-readable format", () => {
    const files: IdeaFile[] = [
      { name: "small.txt", path: "small.txt", type: "file", size: 500 },
      { name: "medium.txt", path: "medium.txt", type: "file", size: 5000 },
    ];

    render(<FileTree files={files} />);

    expect(screen.getByText("500 B")).toBeInTheDocument();
    expect(screen.getByText("4.9 KB")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-004] Test selected file highlighting
  it("highlights selected file", () => {
    const files: IdeaFile[] = [
      { name: "spec.json", path: "spec.json", type: "file", size: 256 },
      { name: "notes.md", path: "notes.md", type: "file", size: 1024 },
    ];

    render(<FileTree files={files} selectedPath="spec.json" />);

    const selectedButton = screen.getByTestId("file-tree-button-spec.json");
    expect(selectedButton).toHaveClass("bg-cyan-500/20");
  });

  // [REQ:REQ-P0-004] Test nested directory structure
  it("renders nested directories correctly", () => {
    const files: IdeaFile[] = [
      {
        name: "docs",
        path: "docs",
        type: "directory",
        children: [
          {
            name: "api",
            path: "docs/api",
            type: "directory",
            children: [
              { name: "schema.json", path: "docs/api/schema.json", type: "file", size: 2048 },
            ],
          },
        ],
      },
    ];

    render(<FileTree files={files} />);

    // First level directory
    expect(screen.getByText("docs")).toBeInTheDocument();

    // Expand first level
    fireEvent.click(screen.getByText("docs"));
    expect(screen.getByText("api")).toBeInTheDocument();

    // Expand second level
    fireEvent.click(screen.getByText("api"));
    expect(screen.getByText("schema.json")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-004] Test directory without children
  it("renders empty directory without expand chevron", () => {
    const files: IdeaFile[] = [
      { name: "empty-dir", path: "empty-dir", type: "directory", children: [] },
    ];

    render(<FileTree files={files} />);

    expect(screen.getByText("empty-dir")).toBeInTheDocument();
    // The expand button should still work (just won't show any children)
    fireEvent.click(screen.getByText("empty-dir"));
  });

  // [REQ:REQ-P0-004] Test custom data-testid
  it("uses custom data-testid when provided", () => {
    const files: IdeaFile[] = [
      { name: "spec.json", path: "spec.json", type: "file", size: 256 },
    ];

    render(<FileTree files={files} data-testid="custom-file-tree" />);

    expect(screen.getByTestId("custom-file-tree")).toBeInTheDocument();
  });
});
