import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CodeBlock } from "./CodeBlock";

const LABELS = { copyLabel: "copy-x", copiedLabel: "copied-x", copyShortLabel: "copy-short-x" };

describe("CodeBlock", () => {
  it("renders code text and language label", () => {
    const lang = "lang-x";
    render(<CodeBlock {...LABELS} data-testid="cb" code="payload-x" language={lang} />);
    const root = screen.getByTestId("cb");
    expect(root.textContent).toContain(lang);
    expect(root.textContent).toContain("payload-x");
  });

  it("copies to clipboard on copy button click", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    render(<CodeBlock {...LABELS} code="payload-x" />);
    await userEvent.click(screen.getByRole("button", { name: LABELS.copyLabel }));
    expect(writeText).toHaveBeenCalledWith("payload-x");
  });

  it("renders line numbers when requested", () => {
    render(<CodeBlock {...LABELS} data-testid="cb" code={"a\nb\nc"} showLineNumbers />);
    const root = screen.getByTestId("cb");
    expect(root.textContent).toContain("1");
    expect(root.textContent).toContain("3");
  });
});
