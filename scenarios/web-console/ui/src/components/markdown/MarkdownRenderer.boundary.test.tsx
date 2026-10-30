import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";

// A renderer failure must never blank a message: the boundary shows the source.
vi.mock("react-markdown", async (original) => ({
  ...await original<typeof import("react-markdown")>(),
  default: () => { throw new Error("parser crashed"); },
}));

const { MarkdownRenderer } = await import("./MarkdownRenderer");

describe("MarkdownRenderer error boundary", () => {
  it("shows the raw markdown when rendering throws", () => {
    const logged = vi.spyOn(console, "error").mockImplementation(() => undefined);
    render(<MarkdownRenderer content={"# Report\n\nKeep the **cutoff**."} />);
    expect(screen.getByText(/# Report/).tagName).toBe("PRE");
    expect(logged).toHaveBeenCalledWith("[MarkdownRenderer] Failed to render content:", expect.any(Error), expect.anything());
    logged.mockRestore();
  });
});
