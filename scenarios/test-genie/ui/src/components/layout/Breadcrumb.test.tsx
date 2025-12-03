import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Breadcrumb } from "./Breadcrumb";

describe("Breadcrumb", () => {
  it("renders all breadcrumb items", () => {
    render(
      <Breadcrumb
        items={[
          { label: "Home" },
          { label: "Settings" },
          { label: "Profile" }
        ]}
      />
    );

    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("Profile")).toBeInTheDocument();
  });

  it("renders clickable items as buttons", () => {
    const onClick = vi.fn();
    render(
      <Breadcrumb
        items={[
          { label: "Home", onClick },
          { label: "Current" }
        ]}
      />
    );

    const homeButton = screen.getByText("Home");
    expect(homeButton.tagName).toBe("BUTTON");

    const currentItem = screen.getByText("Current");
    expect(currentItem.tagName).toBe("SPAN");
  });

  it("calls onClick when clickable item is clicked", () => {
    const onClick = vi.fn();
    render(
      <Breadcrumb
        items={[
          { label: "Home", onClick },
          { label: "Current" }
        ]}
      />
    );

    fireEvent.click(screen.getByText("Home"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("renders separators between items", () => {
    render(
      <Breadcrumb
        items={[
          { label: "Home" },
          { label: "Settings" },
          { label: "Profile" }
        ]}
      />
    );

    // ChevronRight icons should be present (one less than number of items)
    const nav = screen.getByRole("navigation");
    const svgs = nav.querySelectorAll("svg");
    expect(svgs).toHaveLength(2);
  });

  it("does not render separator before first item", () => {
    render(
      <Breadcrumb
        items={[
          { label: "Home" }
        ]}
      />
    );

    const nav = screen.getByRole("navigation");
    const svgs = nav.querySelectorAll("svg");
    expect(svgs).toHaveLength(0);
  });

  it("last item is styled as current (white text)", () => {
    render(
      <Breadcrumb
        items={[
          { label: "Home", onClick: vi.fn() },
          { label: "Current" }
        ]}
      />
    );

    const currentItem = screen.getByText("Current");
    expect(currentItem).toHaveClass("text-white");
  });

  it("handles empty items array", () => {
    render(<Breadcrumb items={[]} />);
    const nav = screen.getByRole("navigation");
    expect(nav.children).toHaveLength(0);
  });
});
