/**
 * EmptyState tests — pin the two optional-slot branches (`description` and
 * `action`), the always-present title + icon, and that the caller className /
 * testId pass through.
 */
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ImageOff } from "lucide-react";

import { EmptyState } from "./empty-state";

describe("EmptyState", () => {
  it("renders the title and the icon, with no description or action by default", () => {
    render(<EmptyState Icon={ImageOff} title="Nothing here" testId="empty" />);

    const root = screen.getByTestId("empty");
    expect(screen.getByText(/Nothing here/)).toBeInTheDocument();
    // Icon is decorative (aria-hidden) and rendered inside the badge span.
    expect(root.querySelector("svg")).not.toBeNull();
    expect(root.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
  });

  it("omits the description paragraph when no description is given", () => {
    render(<EmptyState Icon={ImageOff} title="Empty" testId="empty" />);

    // Only the title paragraph should exist, not a second muted line.
    const paragraphs = screen.getByTestId("empty").querySelectorAll("p");
    expect(paragraphs).toHaveLength(1);
  });

  it("renders the description paragraph when provided", () => {
    render(
      <EmptyState
        Icon={ImageOff}
        title="Empty"
        description="Try dropping an image"
        testId="empty"
      />,
    );

    expect(screen.getByText(/Try dropping an image/)).toBeInTheDocument();
    expect(screen.getByTestId("empty").querySelectorAll("p")).toHaveLength(2);
  });

  it("omits the action wrapper when no action is given", () => {
    render(<EmptyState Icon={ImageOff} title="Empty" testId="empty" />);

    expect(
      screen.getByTestId("empty").querySelector("button"),
    ).toBeNull();
  });

  it("renders the action node when provided", () => {
    render(
      <EmptyState
        Icon={ImageOff}
        title="Empty"
        action={<button type="button">Add one</button>}
        testId="empty"
      />,
    );

    expect(screen.getByRole("button", { name: "Add one" })).toBeInTheDocument();
  });

  it("merges a caller className onto the root", () => {
    render(<EmptyState Icon={ImageOff} title="Empty" className="custom-cls" testId="empty" />);

    expect(screen.getByTestId("empty")).toHaveClass("custom-cls");
  });
});
