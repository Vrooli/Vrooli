import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("shiki", () => ({
  createHighlighter: vi.fn().mockResolvedValue({
    getLoadedLanguages: () => ["typescript"],
    codeToHtml: vi
      .fn()
      .mockResolvedValue("<pre><code>highlighted</code></pre>"),
  }),
}));

import { CodeBlock } from "@vrooli/react-component-library/markdown-renderer/0.4.2";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

// Library-owned copy and mock output, named once so the assertions survive a
// wording change in one place.
const languageLabel = "TYPESCRIPT";
const highlightedMarkup = "highlighted";
const codeSource = "const ready = true";

describe("CodeBlock", () => {
  // Restore the whole `navigator` object, not just mocks: replacing it with a
  // clipboard-only stub would otherwise leak into every later file in the
  // single-threaded test run.
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("shows a language label, upgrades to highlighted HTML, and copies source", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    renderWithProviders(<CodeBlock code={codeSource} language="ts" />);

    expect(screen.getByText(languageLabel)).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByText(highlightedMarkup)).toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(codeSource));
  });
});
