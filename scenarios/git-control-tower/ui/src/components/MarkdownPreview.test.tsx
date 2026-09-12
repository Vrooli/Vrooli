import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MarkdownPreview } from "./MarkdownPreview";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("MarkdownPreview", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders GFM table markdown as an HTML table", () => {
    const content = `
| Name | Status |
| --- | --- |
| API | Ready |
| UI | In Progress |
`;

    renderWithProviders(<MarkdownPreview content={content} />);

    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Name" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Status" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "Ready" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "In Progress" })).toBeInTheDocument();
  });

  it("renders fenced code blocks with language header and copy action", () => {
    const content = `
\`\`\`ts
const value: number = 42;
\`\`\`
`;

    renderWithProviders(<MarkdownPreview content={content} />);

    expect(screen.getByText("typescript")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy code" })).toBeInTheDocument();
  });
});
