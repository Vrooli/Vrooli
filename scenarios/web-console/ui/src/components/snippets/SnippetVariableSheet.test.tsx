import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils";
import type { SnippetDTO } from "../../api/snippets";
import { SnippetVariableSheet } from "./SnippetVariableSheet";

const snippet: SnippetDTO = { id: "s", name: "Probe", body: "Check {{scenario}} with {{owner}}", color: "", pinned: false, use_count: 0, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" };

describe("SnippetVariableSheet", () => {
  it("shows unresolved tokens verbatim in the live preview", () => {
    render(<SnippetVariableSheet open snippet={snippet} autoValues={{}} onClose={vi.fn()} onInsert={vi.fn()} />);
    expect(screen.getByTestId("snippet-variable-preview")).toHaveTextContent("Check {{scenario}} with {{owner}}");
    expect(screen.getAllByRole("textbox")).toHaveLength(2);
  });

  it("renders an auto-resolved name read-only with no matching input", () => {
    render(<SnippetVariableSheet open snippet={snippet} autoValues={{ scenario: "web-console" }} onClose={vi.fn()} onInsert={vi.fn()} />);
    expect(screen.getByTestId("snippet-variable-readonly-scenario")).toHaveTextContent("web-console");
    expect(screen.queryByTestId("snippet-variable-input-scenario")).toBeNull();
    expect(screen.getByTestId("snippet-variable-input-owner")).toBeTruthy();
  });
});
