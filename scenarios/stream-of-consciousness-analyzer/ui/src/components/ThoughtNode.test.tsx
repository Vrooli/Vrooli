import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ThoughtNode } from "./ThoughtNode";
import type { Thought } from "../lib/types";

function makeMockThought(overrides: Partial<Thought> = {}): Thought {
  return {
    id: "t1",
    title: "Test Thought",
    body: "A body",
    scheme_id: "s1",
    canvas_x: 10,
    canvas_y: 20,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("ThoughtNode", () => {
  // [REQ:P0-004] Test thought node renders title and body
  it("renders title and body", () => {
    render(
      <ThoughtNode
        thought={makeMockThought()}
        isSource={false}
        isLinkMode={false}
        onClick={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText("Test Thought")).toBeTruthy();
    expect(screen.getByText("A body")).toBeTruthy();
  });

  // [REQ:P0-004] Test thought node shows "Untitled" for empty title
  it("shows Untitled for empty title", () => {
    render(
      <ThoughtNode
        thought={makeMockThought({ title: "" })}
        isSource={false}
        isLinkMode={false}
        onClick={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText("Untitled")).toBeTruthy();
  });

  // [REQ:P0-004] Test thought node click calls onClick
  it("calls onClick when clicked", () => {
    const onClick = vi.fn();
    render(
      <ThoughtNode
        thought={makeMockThought()}
        isSource={false}
        isLinkMode={false}
        onClick={onClick}
        onDelete={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByTestId("thought-node"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  // [REQ:P0-004] Test thought node delete button calls onDelete
  it("calls onDelete and stops propagation", () => {
    const onClick = vi.fn();
    const onDelete = vi.fn();
    render(
      <ThoughtNode
        thought={makeMockThought()}
        isSource={false}
        isLinkMode={false}
        onClick={onClick}
        onDelete={onDelete}
      />,
    );
    fireEvent.click(screen.getByLabelText("Delete thought: Test Thought"));
    expect(onDelete).toHaveBeenCalledOnce();
    expect(onClick).not.toHaveBeenCalled();
  });

  // [REQ:P0-004] Test source styling
  it("applies source styling when isSource is true", () => {
    render(
      <ThoughtNode
        thought={makeMockThought()}
        isSource={true}
        isLinkMode={true}
        onClick={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    const node = screen.getByTestId("thought-node");
    expect(node.className).toContain("border-blue-500");
  });

  // [REQ:P0-004] Test link mode styling when not source
  it("applies link mode styling when isLinkMode but not isSource", () => {
    render(
      <ThoughtNode
        thought={makeMockThought()}
        isSource={false}
        isLinkMode={true}
        onClick={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    const node = screen.getByTestId("thought-node");
    expect(node.className).toContain("hover:border-blue-400");
  });
});
