import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { DiffViewer } from "./DiffViewer";

describe("DiffViewer", () => {
  it("renders the file path header", () => {
    render(<DiffViewer before="a" after="b" filePath="scenarios/foo/Makefile" />);
    expect(screen.getByText("scenarios/foo/Makefile")).toBeInTheDocument();
  });

  it("renders added lines with + marker", () => {
    const { container } = render(<DiffViewer before="" after="new line" filePath="test.txt" />);
    expect(container.querySelector(".text-green-400")).toBeTruthy();
    expect(screen.getByText("+")).toBeInTheDocument();
    expect(screen.getByText("new line")).toBeInTheDocument();
  });

  it("renders removed lines with minus marker", () => {
    const { container } = render(<DiffViewer before="old line" after="" filePath="test.txt" />);
    expect(container.querySelector(".text-red-400")).toBeTruthy();
    expect(screen.getByText("\u2212")).toBeInTheDocument();
    expect(screen.getByText("old line")).toBeInTheDocument();
  });

  it("renders unchanged lines without color", () => {
    render(<DiffViewer before="same" after="same" filePath="test.txt" />);
    expect(screen.getByText("same")).toBeInTheDocument();
  });

  it("renders mixed changes correctly", () => {
    const { container } = render(
      <DiffViewer before="keep\nremove" after="keep\nadd" filePath="test.txt" />
    );
    const html = container.innerHTML;
    // Should contain content for: unchanged("keep"), removed("remove"), added("add")
    expect(html).toContain("keep");
    expect(html).toContain("remove");
    expect(html).toContain("add");
    // Should have both red (removed) and green (added) styling
    expect(html).toContain("bg-red-500/10");
    expect(html).toContain("bg-green-500/10");
  });
});
