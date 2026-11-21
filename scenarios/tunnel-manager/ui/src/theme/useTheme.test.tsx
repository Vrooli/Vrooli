import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useTheme } from "./useTheme";

describe("useTheme", () => {
  it("rejects use outside ThemeProvider", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => useTheme())).toThrow("useTheme must be called inside <ThemeProvider>");
    consoleError.mockRestore();
  });
});
