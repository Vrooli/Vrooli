import { describe, it, expect, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import * as React from "react";
import { Panel } from "./panel";

afterEach(cleanup);

const PANEL_TITLE = "My Panel";
const BODY_TEXT = "Body content";
const DESC_TEXT = "Helpful info";

describe("Panel", () => {
  it("renders the title", () => {
    render(<Panel title={PANEL_TITLE}><p>{BODY_TEXT}</p></Panel>);
    expect(screen.getByText(PANEL_TITLE)).toBeInTheDocument();
  });

  it("renders children in the body slot", () => {
    render(<Panel title="T"><p>{BODY_TEXT}</p></Panel>);
    expect(screen.getByText(BODY_TEXT)).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(<Panel title="T" description={DESC_TEXT}><p>body</p></Panel>);
    expect(screen.getByText(DESC_TEXT)).toBeInTheDocument();
  });

  it("does not render description element when omitted", () => {
    render(<Panel title="T"><p>body</p></Panel>);
    expect(screen.queryByText(DESC_TEXT)).not.toBeInTheDocument();
  });

  it("renders actions when provided", () => {
    render(<Panel title="T" actions={<button>Action</button>}><p>body</p></Panel>);
    expect(screen.getByRole("button", { name: "Action" })).toBeInTheDocument();
  });

  it("does not render actions wrapper when omitted", () => {
    render(<Panel title="T"><p>body</p></Panel>);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("applies padding when bodyless is false (default)", () => {
    const { container } = render(<Panel title="T"><p>body</p></Panel>);
    const bodyDiv = container.querySelector("section > div:last-child");
    expect(bodyDiv).toHaveClass("p-4");
  });

  it("removes padding when bodyless is true", () => {
    const { container } = render(<Panel title="T" bodyless><p>body</p></Panel>);
    const bodyDiv = container.querySelector("section > div:last-child");
    expect(bodyDiv).not.toHaveClass("p-4");
  });

  it("applies extra className to the section", () => {
    const { container } = render(<Panel title="T" className="extra-class"><p>body</p></Panel>);
    expect(container.firstChild).toHaveClass("extra-class");
  });

  it("forwards ref to the section element", () => {
    const ref = React.createRef<HTMLElement>();
    render(<Panel title="T" ref={ref}><p>body</p></Panel>);
    expect(ref.current?.tagName).toBe("SECTION");
  });
});
