import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { DocTree, type DocTreeProps } from "./DocTree";
import type { DocTreeNode } from "../../../shared/services/documentationApi";

const tree: DocTreeNode = {
  name: "alpha",
  path: "scenarios/alpha",
  type: "directory",
  children: [
    {
      name: "README.md",
      path: "scenarios/alpha/README.md",
      type: "file",
      doc_type: "readme",
      warning: {
        type: "misplaced",
        message: "Documentation file is in the wrong location",
        severity: "warning",
      },
    },
  ],
};

const createProps = (overrides: Partial<DocTreeProps> = {}): DocTreeProps => ({
  tree,
  selectedPath: null,
  onSelectPath: vi.fn(),
  isLoading: false,
  hasError: false,
  errorMessage: "",
  onRefresh: vi.fn(),
  ...overrides,
});

describe("DocTree", () => {
  it("renders tree nodes and warnings", () => {
    render(<DocTree {...createProps()} />);

    expect(screen.getByText("README.md")).toBeDefined();
    expect(screen.getByText("misplaced")).toBeDefined();
  });

  it("renders fallback when no tree", () => {
    render(<DocTree {...createProps({ tree: undefined })} />);

    expect(screen.getByText(/Select a scenario/i)).toBeDefined();
  });
});
