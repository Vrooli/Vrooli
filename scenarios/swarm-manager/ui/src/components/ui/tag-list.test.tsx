import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { TagList } from "./tag-list";

/**
 * TagList Component Tests
 *
 * Tests verify the truncation and display logic:
 * - Empty/null arrays return nothing
 * - Tags at or under threshold show all tags
 * - Tags over threshold show truncated list with "+N" indicator
 * - Custom maxTags prop controls truncation
 * - Custom className is applied
 *
 * [REQ:PHASE12] Unified utility component tests
 */

describe("TagList", () => {
  describe("empty state handling", () => {
    it("returns null for empty tags array", () => {
      const { container } = render(<TagList tags={[]} />);

      expect(container.firstChild).toBeNull();
    });

    it("returns null for undefined tags array", () => {
      // @ts-expect-error - testing runtime behavior for undefined
      const { container } = render(<TagList tags={undefined} />);

      expect(container.firstChild).toBeNull();
    });
  });

  describe("tag display without truncation", () => {
    it("renders single tag", () => {
      render(<TagList tags={["react"]} />);

      expect(screen.getByText("react")).toBeInTheDocument();
    });

    it("renders multiple tags under default threshold", () => {
      render(<TagList tags={["react", "typescript", "vite"]} />);

      expect(screen.getByText("react")).toBeInTheDocument();
      expect(screen.getByText("typescript")).toBeInTheDocument();
      expect(screen.getByText("vite")).toBeInTheDocument();
    });

    it("does not show +N indicator when tags equal maxTags", () => {
      render(<TagList tags={["one", "two", "three"]} maxTags={3} />);

      expect(screen.queryByText(/\+\d/)).not.toBeInTheDocument();
    });

    it("does not show +N indicator when tags are under maxTags", () => {
      render(<TagList tags={["one", "two"]} maxTags={3} />);

      expect(screen.queryByText(/\+\d/)).not.toBeInTheDocument();
    });
  });

  describe("tag truncation", () => {
    it("shows +N indicator when tags exceed default maxTags (3)", () => {
      render(
        <TagList tags={["react", "typescript", "vite", "tailwind", "shadcn"]} />
      );

      // First 3 visible
      expect(screen.getByText("react")).toBeInTheDocument();
      expect(screen.getByText("typescript")).toBeInTheDocument();
      expect(screen.getByText("vite")).toBeInTheDocument();

      // Hidden tags
      expect(screen.queryByText("tailwind")).not.toBeInTheDocument();
      expect(screen.queryByText("shadcn")).not.toBeInTheDocument();

      // +2 indicator (5 - 3 = 2 hidden)
      expect(screen.getByText("+2")).toBeInTheDocument();
    });

    it("shows +1 indicator for one hidden tag", () => {
      render(<TagList tags={["a", "b", "c", "d"]} maxTags={3} />);

      expect(screen.getByText("+1")).toBeInTheDocument();
    });

    it("respects custom maxTags prop", () => {
      render(
        <TagList tags={["a", "b", "c", "d", "e"]} maxTags={2} />
      );

      // First 2 visible
      expect(screen.getByText("a")).toBeInTheDocument();
      expect(screen.getByText("b")).toBeInTheDocument();

      // Rest hidden
      expect(screen.queryByText("c")).not.toBeInTheDocument();
      expect(screen.queryByText("d")).not.toBeInTheDocument();
      expect(screen.queryByText("e")).not.toBeInTheDocument();

      // +3 indicator (5 - 2 = 3 hidden)
      expect(screen.getByText("+3")).toBeInTheDocument();
    });

    it("shows all tags when maxTags is larger than array length", () => {
      render(<TagList tags={["a", "b"]} maxTags={10} />);

      expect(screen.getByText("a")).toBeInTheDocument();
      expect(screen.getByText("b")).toBeInTheDocument();
      expect(screen.queryByText(/\+\d/)).not.toBeInTheDocument();
    });
  });

  describe("styling", () => {
    it("applies custom className to container", () => {
      render(<TagList tags={["test"]} className="custom-class mt-4" />);

      const container = screen.getByText("test").parentElement;
      expect(container).toHaveClass("custom-class");
      expect(container).toHaveClass("mt-4");
    });

    it("applies default flex and gap classes", () => {
      render(<TagList tags={["test"]} />);

      const container = screen.getByText("test").parentElement;
      expect(container).toHaveClass("flex");
      expect(container).toHaveClass("flex-wrap");
      expect(container).toHaveClass("gap-1");
    });

    it("renders tags with proper styling classes", () => {
      render(<TagList tags={["styled-tag"]} />);

      const tag = screen.getByText("styled-tag");
      expect(tag).toHaveClass("rounded-full");
      expect(tag).toHaveClass("text-xs");
    });
  });

  describe("edge cases", () => {
    it("handles maxTags of 0 (shows only +N)", () => {
      render(<TagList tags={["a", "b", "c"]} maxTags={0} />);

      expect(screen.queryByText("a")).not.toBeInTheDocument();
      expect(screen.queryByText("b")).not.toBeInTheDocument();
      expect(screen.queryByText("c")).not.toBeInTheDocument();
      expect(screen.getByText("+3")).toBeInTheDocument();
    });

    it("handles maxTags of 1", () => {
      render(<TagList tags={["only-this", "hidden"]} maxTags={1} />);

      expect(screen.getByText("only-this")).toBeInTheDocument();
      expect(screen.queryByText("hidden")).not.toBeInTheDocument();
      expect(screen.getByText("+1")).toBeInTheDocument();
    });

    it("handles tags with special characters", () => {
      render(<TagList tags={["c++", "c#", "node.js"]} />);

      expect(screen.getByText("c++")).toBeInTheDocument();
      expect(screen.getByText("c#")).toBeInTheDocument();
      expect(screen.getByText("node.js")).toBeInTheDocument();
    });

    it("handles tags with spaces", () => {
      render(<TagList tags={["react query", "tanstack router"]} />);

      expect(screen.getByText("react query")).toBeInTheDocument();
      expect(screen.getByText("tanstack router")).toBeInTheDocument();
    });

    it("handles large number of hidden tags", () => {
      const manyTags = Array.from({ length: 100 }, (_, i) => `tag-${i}`);
      render(<TagList tags={manyTags} maxTags={3} />);

      expect(screen.getByText("tag-0")).toBeInTheDocument();
      expect(screen.getByText("tag-1")).toBeInTheDocument();
      expect(screen.getByText("tag-2")).toBeInTheDocument();
      expect(screen.getByText("+97")).toBeInTheDocument();
    });
  });
});
