import { render, screen, fireEvent } from "@testing-library/react";
import { beforeEach, describe, it, expect } from "vitest";
import { Info } from "lucide-react";
import { DetailSection } from "./DetailSection";

describe("DetailSection", () => {
  it("renders title and children", () => {
    render(
      <DetailSection title="Test Section">
        <p>Section content</p>
      </DetailSection>,
    );
    expect(screen.getByText("Test Section")).toBeInTheDocument();
    expect(screen.getByText("Section content")).toBeInTheDocument();
  });

  it("renders top divider by default", () => {
    const { container } = render(
      <DetailSection title="With Divider">
        <p>Content</p>
      </DetailSection>,
    );
    const section = container.querySelector("section");
    expect(section).toHaveClass("border-t");
    expect(section).toHaveClass("mt-4");
    expect(section).toHaveClass("pt-4");
  });

  it("hides divider when hideDivider is true", () => {
    const { container } = render(
      <DetailSection title="No Divider" hideDivider>
        <p>Content</p>
      </DetailSection>,
    );
    const section = container.querySelector("section");
    expect(section).not.toHaveClass("border-t");
    expect(section).toHaveClass("pt-1");
  });

  it("renders icon when provided", () => {
    const { container } = render(
      <DetailSection title="With Icon" icon={Info}>
        <p>Content</p>
      </DetailSection>,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders action slot", () => {
    render(
      <DetailSection title="With Action" action={<button type="button">Edit</button>}>
        <p>Content</p>
      </DetailSection>,
    );
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });

  it("forwards data-testid", () => {
    render(
      <DetailSection title="Testable" data-testid="my-section">
        <p>Content</p>
      </DetailSection>,
    );
    expect(screen.getByTestId("my-section")).toBeInTheDocument();
  });

  describe("collapsible (storageKey)", () => {
    beforeEach(() => {
      window.localStorage.clear();
    });

    it("toggles content and persists state per storageKey", () => {
      render(
        <DetailSection title="Collapsible" storageKey="test.section" data-testid="collapsible-section">
          <p>Hidden treasure</p>
        </DetailSection>,
      );

      // Open by default.
      expect(screen.getByText("Hidden treasure")).toBeInTheDocument();
      const toggle = screen.getByTestId("collapsible-section-toggle");
      expect(toggle).toHaveAttribute("aria-expanded", "true");

      fireEvent.click(toggle);
      expect(screen.queryByText("Hidden treasure")).toBeNull();
      expect(toggle).toHaveAttribute("aria-expanded", "false");
      expect(window.localStorage.getItem("swarm-manager.section.test.section")).toBe("0");
    });

    it("respects defaultOpen false", () => {
      render(
        <DetailSection title="Closed" storageKey="test.closed" defaultOpen={false}>
          <p>Not yet</p>
        </DetailSection>,
      );
      expect(screen.queryByText("Not yet")).toBeNull();
    });
  });
});
