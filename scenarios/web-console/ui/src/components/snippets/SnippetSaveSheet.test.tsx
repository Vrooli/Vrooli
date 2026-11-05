import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils";
import { HEADER_COLORS } from "../../consts/config";

const hook = vi.hoisted(() => ({ useSnippets: vi.fn() }));
vi.mock("../../hooks/useSnippets", () => hook);

import { defaultSnippetName, SnippetSaveSheet } from "./SnippetSaveSheet";

describe("SnippetSaveSheet", () => {
  const save = vi.fn();
  beforeEach(() => {
    vi.clearAllMocks();
    save.mockResolvedValue({ id: "saved" });
    hook.useSnippets.mockReturnValue({ save });
  });

  it("prefills a readable word-boundary name and removes line markers", () => {
    expect(defaultSnippetName("Before editing, name the seam that owns this behaviour and show me the grep."))
      .toBe("Before editing, name the seam that owns");
    expect(defaultSnippetName("- Check the process")).toBe("Check the process");
    expect(defaultSnippetName("## Demand evidence")).toBe("Demand evidence");
  });

  it("renders exactly eight 44px swatches and selects a valid caller colour", () => {
    render(<SnippetSaveSheet open onClose={vi.fn()} mode="create" initialBody="body" initialColor={HEADER_COLORS[3]} />);
    const swatches = HEADER_COLORS.map((color) => screen.getByTestId(`snippet-color-${color}`));
    expect(swatches).toHaveLength(8);
    expect(swatches.every((swatch) => swatch.className.includes("h-11") && swatch.className.includes("w-11"))).toBe(true);
    expect(swatches[3]).toHaveAttribute("aria-pressed", "true");
  });

  it("falls an unknown colour back to the first swatch", () => {
    render(<SnippetSaveSheet open onClose={vi.fn()} mode="create" initialBody="body" initialColor="#000000" />);
    expect(screen.getByTestId(`snippet-color-${HEADER_COLORS[0]}`)).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByTestId("snippet-color-#000000")).toBeNull();
  });

  it("wraps a selected phrase with a valid variable and updates the real count", () => {
    vi.spyOn(window, "prompt").mockReturnValue("scenario");
    render(<SnippetSaveSheet open onClose={vi.fn()} mode="create" initialBody="Check web-console first" />);
    const body = screen.getByTestId<HTMLTextAreaElement>("snippet-save-body");
    body.setSelectionRange(6, 17);
    fireEvent.select(body);
    fireEvent.click(screen.getByTestId("snippet-make-variable"));
    expect(body.value).toBe("Check {{scenario}} first");
    expect(screen.getByTestId("snippet-variable-count")).toHaveTextContent("snippets.variableCount");
  });

  it("rejects an invalid name without changing the body", () => {
    vi.spyOn(window, "prompt").mockReturnValue("Scenario");
    render(<SnippetSaveSheet open onClose={vi.fn()} mode="create" initialBody="Check web-console" />);
    const body = screen.getByTestId<HTMLTextAreaElement>("snippet-save-body");
    body.setSelectionRange(6, 17);
    fireEvent.select(body);
    fireEvent.click(screen.getByTestId("snippet-make-variable"));
    expect(body.value).toBe("Check web-console");
    expect(screen.getByTestId("snippet-variable-error")).toHaveTextContent("snippets.variables.invalidName");
  });

  it("keeps all input when saving fails", async () => {
    save.mockRejectedValue(new Error("offline"));
    render(<SnippetSaveSheet open onClose={vi.fn()} mode="create" initialBody="keep body" />);
    fireEvent.change(screen.getByTestId("snippet-save-name"), { target: { value: "keep name" } });
    fireEvent.click(screen.getByTestId("snippet-save-submit"));
    await waitFor(() => { expect(screen.getByTestId("snippet-save-error")).toBeTruthy(); });
    expect(screen.getByTestId("snippet-save-name")).toHaveValue("keep name");
    expect(screen.getByTestId("snippet-save-body")).toHaveValue("keep body");
  });
});
