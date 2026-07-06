import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import * as React from "react";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { Card, CardHeader, CardTitle, CardDescription } from "./card";

afterEach(cleanup);

const CARD_TEXT = "Hello";
const TITLE_TEXT = "My Title";
const DESC_TEXT = "Some description";
const HEADER_TEXT = "Header";

describe("Card", () => {
  it("renders children", () => {
    render(<Card>{CARD_TEXT}</Card>);
    expect(screen.getByText(CARD_TEXT)).toBeInTheDocument();
  });

  it("applies default variant (border-app-border)", () => {
    const { container } = render(<Card>Content</Card>);
    expect(container.firstChild).toHaveClass("border-app-border");
  });

  it("applies raised variant", () => {
    const { container } = render(<Card variant="raised">Content</Card>);
    expect(container.firstChild).toHaveClass("shadow-sm");
  });

  it("applies danger variant", () => {
    const { container } = render(<Card variant="danger">Content</Card>);
    expect(container.firstChild).toHaveClass("bg-app-danger-soft");
  });

  it("applies muted variant", () => {
    const { container } = render(<Card variant="muted">Content</Card>);
    expect(container.firstChild).toHaveClass("bg-app-surface-muted");
  });

  it("applies padding=none", () => {
    const { container } = render(<Card padding="none">Content</Card>);
    expect(container.firstChild).not.toHaveClass("p-3");
    expect(container.firstChild).not.toHaveClass("p-4");
    expect(container.firstChild).not.toHaveClass("p-6");
  });

  it("applies padding=sm", () => {
    const { container } = render(<Card padding="sm">Content</Card>);
    expect(container.firstChild).toHaveClass("p-3");
  });

  it("applies padding=lg", () => {
    const { container } = render(<Card padding="lg">Content</Card>);
    expect(container.firstChild).toHaveClass("p-6");
  });

  it("merges additional className", () => {
    const { container } = render(<Card className="my-custom">Content</Card>);
    expect(container.firstChild).toHaveClass("my-custom");
  });

  it("forwards ref", () => {
    const ref = { current: null } as React.RefObject<HTMLDivElement>;
    render(<Card ref={ref}>Content</Card>);
    expect(ref.current).not.toBeNull();
  });
});

describe("CardHeader", () => {
  it("renders children", () => {
    render(<CardHeader><span>{HEADER_TEXT}</span></CardHeader>);
    expect(screen.getByText(HEADER_TEXT)).toBeInTheDocument();
  });

  it("applies additional className", () => {
    const { container } = render(<CardHeader className="extra">H</CardHeader>);
    expect(container.firstChild).toHaveClass("extra");
  });
});

describe("CardTitle", () => {
  it("renders an h3 with the text", () => {
    render(<CardTitle>{TITLE_TEXT}</CardTitle>);
    const heading = screen.getByRole("heading", { level: 3 });
    expect(heading).toHaveTextContent(TITLE_TEXT);
  });

  it("applies additional className", () => {
    render(<CardTitle className="bold-title">{TITLE_TEXT}</CardTitle>);
    expect(screen.getByRole("heading")).toHaveClass("bold-title");
  });
});

describe("CardDescription", () => {
  it("renders description text", () => {
    render(<CardDescription>{DESC_TEXT}</CardDescription>);
    expect(screen.getByText(DESC_TEXT)).toBeInTheDocument();
  });

  it("applies additional className", () => {
    const { container } = render(<CardDescription className="desc-class">D</CardDescription>);
    expect(container.firstChild).toHaveClass("desc-class");
  });
});
