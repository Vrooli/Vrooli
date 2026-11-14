import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

import { ThemeContext, useTheme, type ThemeContextValue } from "./themeContext";

describe("useTheme", () => {
  it("gives the provider's choice and setter to any component below it", () => {
    const value: ThemeContextValue = { choice: "system", resolved: "dark", setTheme: vi.fn() };
    const { result } = renderHook(() => useTheme(), {
      wrapper: ({ children }) => <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>,
    });
    expect(result.current).toBe(value);
    result.current.setTheme("light");
    expect(value.setTheme).toHaveBeenCalledWith("light");
  });

  it("fails loudly, saying what is missing, when used outside the provider", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    expect(() => renderHook(() => useTheme())).toThrow("useTheme must be called inside <ThemeProvider>");
    vi.mocked(console.error).mockRestore();
  });
});
