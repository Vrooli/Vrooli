import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { DocSearchPanel } from "./DocSearchPanel";

const baseProps = {
  mode: "files" as const,
  pattern: "",
  query: "",
  scope: "global",
  scenario: "",
  basePath: "",
  includeContent: false,
  fileTypes: "md",
  caseSensitive: false,
  contextLines: 1,
  useSemantic: true,
  isLoading: false,
  hasError: false,
  errorMessage: "",
  hasData: false,
  hasResults: false,
  isSubmitDisabled: false,
  isClearDisabled: true,
  displayQuery: "",
  totalResults: 0,
  tookMsLabel: "?ms",
  results: [],
  onPatternChange: () => undefined,
  onQueryChange: () => undefined,
  onScopeChange: () => undefined,
  onScenarioChange: () => undefined,
  onBasePathChange: () => undefined,
  onIncludeContentChange: () => undefined,
  onFileTypesChange: () => undefined,
  onCaseSensitiveChange: () => undefined,
  onContextLinesChange: () => undefined,
  onUseSemanticChange: () => undefined,
  onSubmit: () => undefined,
  onClear: () => undefined,
};

describe("DocSearchPanel", () => {
  it("shows pattern input for file mode", () => {
    render(<DocSearchPanel {...baseProps} mode="files" />);
    expect(screen.getByPlaceholderText("**/README.md")).toBeTruthy();
    expect(screen.queryByPlaceholderText("health score")).toBeNull();
  });

  it("shows query input for text mode", () => {
    render(<DocSearchPanel {...baseProps} mode="text" />);
    expect(screen.getByPlaceholderText("health score")).toBeTruthy();
    expect(screen.queryByPlaceholderText("**/README.md")).toBeNull();
  });

  it("shows both query and pattern inputs for unified mode", () => {
    render(<DocSearchPanel {...baseProps} mode="unified" />);
    expect(screen.getByPlaceholderText("search documentation by topic or filename")).toBeTruthy();
    expect(screen.getByPlaceholderText("**/README.md")).toBeTruthy();
  });
});
