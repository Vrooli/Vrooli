import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@vrooli/react-component-library/Card/1.3.1";

afterEach(cleanup);

const CARD_TEXT = "Hello";
const TITLE_TEXT = "My Title";
const DESC_TEXT = "Some description";
describe("Card", () => {
  it("renders children", () => {
    render(<Card>{CARD_TEXT}</Card>);
    expect(screen.getByText(CARD_TEXT)).toBeInTheDocument();
  });

  it("merges additional className", () => {
    const { container } = render(<Card className="my-custom">Content</Card>);
    expect(container.firstChild).toHaveClass("rcl-card", "my-custom");
  });

  it("marks the shared surface and keeps an explicit test id", () => {
    render(<Card data-testid="usage-card">Content</Card>);
    expect(screen.getByTestId("usage-card")).toHaveAttribute("data-rcl-card");
  });
});

describe("CardHeader", () => {
  it("renders children", () => {
    const { container } = render(<CardHeader><span>Header</span></CardHeader>);
    expect(container.firstChild).toHaveTextContent("Header");
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

  it("supports the heading level required by the page outline", () => {
    render(<CardTitle as="h2">Section title</CardTitle>);
    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Section title");
  });
});

describe("CardContent", () => {
  it("provides the shared body region", () => {
    const { container } = render(<CardContent>Body</CardContent>);
    expect(container.firstChild).toHaveClass("rcl-card__content");
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
