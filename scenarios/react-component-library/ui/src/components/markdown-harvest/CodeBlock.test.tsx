import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("shiki", () => ({
  createHighlighter: vi.fn().mockResolvedValue({
    getLoadedLanguages: () => ["typescript"],
    codeToHtml: vi.fn().mockResolvedValue("<pre><code>highlighted</code></pre>"),
  }),
}));

import { CodeBlock } from "./CodeBlock";
import { renderWithProviders } from "../../test-utils";

describe("CodeBlock", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows a language label, upgrades to highlighted HTML, and copies source", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    renderWithProviders(<CodeBlock code="const ready = true" language="ts" />);

    expect(screen.getByText("TYPESCRIPT")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("highlighted")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("const ready = true"));
  });
});
