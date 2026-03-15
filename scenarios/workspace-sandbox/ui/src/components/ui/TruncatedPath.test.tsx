import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TruncatedPath } from "./TruncatedPath";

describe("TruncatedPath", () => {
  it("renders a short path without truncation", () => {
    render(<TruncatedPath path="/a/b" maxLength={30} />);
    expect(screen.getByTestId("truncated-path")).toHaveTextContent("/a/b");
  });

  it("renders a long path truncated from the left", () => {
    const longPath = "/home/user/Vrooli/scenarios/web-console";
    render(<TruncatedPath path={longPath} maxLength={25} />);
    const el = screen.getByTestId("truncated-path");
    expect(el.textContent).toMatch(/^…\//);
    expect(el.textContent).toContain("web-console");
  });

  it("expands on click to show full path", () => {
    const longPath = "/home/user/Vrooli/scenarios/web-console";
    render(<TruncatedPath path={longPath} maxLength={25} />);
    const el = screen.getByTestId("truncated-path");

    fireEvent.click(el);
    expect(el).toHaveTextContent(longPath);
  });

  it("collapses on second click", () => {
    const longPath = "/home/user/Vrooli/scenarios/web-console";
    render(<TruncatedPath path={longPath} maxLength={25} />);
    const el = screen.getByTestId("truncated-path");

    fireEvent.click(el);
    expect(el).toHaveTextContent(longPath);

    fireEvent.click(el);
    expect(el.textContent).toMatch(/^…\//);
  });

  it("applies font-mono class when mono prop is true (default)", () => {
    render(<TruncatedPath path="/a/b" />);
    expect(screen.getByTestId("truncated-path").className).toContain("font-mono");
  });

  it("does not apply font-mono when mono prop is false", () => {
    render(<TruncatedPath path="/a/b" mono={false} />);
    expect(screen.getByTestId("truncated-path").className).not.toContain("font-mono");
  });

  it("does not show cursor-pointer for non-truncated paths", () => {
    render(<TruncatedPath path="/a/b" maxLength={30} />);
    expect(screen.getByTestId("truncated-path").className).not.toContain("cursor-pointer");
  });
});
