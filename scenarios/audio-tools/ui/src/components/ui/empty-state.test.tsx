import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { EmptyState } from "./empty-state";

afterEach(cleanup);

const TITLE_TEXT = "Nothing here";
const DESC_TEXT = "Some detail";

describe("EmptyState", () => {
  it("renders the title", () => {
    render(<EmptyState title={TITLE_TEXT} />);
    expect(screen.getByText(TITLE_TEXT)).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(<EmptyState title="T" description={DESC_TEXT} />);
    expect(screen.getByText(DESC_TEXT)).toBeInTheDocument();
  });

  it("does not render description element when omitted", () => {
    render(<EmptyState title="T" />);
    expect(screen.queryByText(DESC_TEXT)).not.toBeInTheDocument();
  });

  it("renders icon when provided", () => {
    render(<EmptyState title="T" icon={<span data-testid="icon">icon</span>} />);
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  it("does not render icon slot when icon is omitted", () => {
    const { container } = render(<EmptyState title="T" />);
    // only the outer wrapper should be present (no icon child div)
    const divs = container.querySelectorAll("div");
    expect(divs.length).toBe(1);
  });

  it("renders action when provided", () => {
    render(<EmptyState title="T" action={<button>Do it</button>} />);
    expect(screen.getByRole("button", { name: "Do it" })).toBeInTheDocument();
  });

  it("does not render action slot when action is omitted", () => {
    render(<EmptyState title="T" />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("default tone=neutral: no role=alert", () => {
    render(<EmptyState title="T" />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("tone=error: renders role=alert", () => {
    const ERROR_TITLE = "Error!";
    render(<EmptyState title={ERROR_TITLE} tone="error" />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("applies extra className", () => {
    const { container } = render(<EmptyState title="T" className="custom" />);
    expect(container.firstChild).toHaveClass("custom");
  });
});
