import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { useUrlState } from "./use-url-state";

function wrapper(initialEntries: string[] = ["/"]) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>;
  };
}

describe("useUrlState", () => {
  it("returns default value when param is absent", () => {
    const { result } = renderHook(() => useUrlState<string>("tab", "all"), {
      wrapper: wrapper(),
    });
    expect(result.current[0]).toBe("all");
  });

  it("reads initial value from URL", () => {
    const { result } = renderHook(() => useUrlState<string>("tab", "all"), {
      wrapper: wrapper(["/?tab=fix"]),
    });
    expect(result.current[0]).toBe("fix");
  });

  it("updates URL when setter is called", () => {
    const { result } = renderHook(() => useUrlState<string>("tab", "all"), {
      wrapper: wrapper(),
    });
    act(() => {
      result.current[1]("fix");
    });
    expect(result.current[0]).toBe("fix");
  });

  it("removes param from URL when set to default", () => {
    const { result } = renderHook(() => useUrlState<string>("tab", "all"), {
      wrapper: wrapper(["/?tab=fix"]),
    });
    expect(result.current[0]).toBe("fix");
    act(() => {
      result.current[1]("all");
    });
    expect(result.current[0]).toBe("all");
  });

  it("falls back to default for invalid values when validate is provided", () => {
    const isValid = (v: string): v is "a" | "b" => v === "a" || v === "b";
    const { result } = renderHook(
      () => useUrlState("x", "a", { validate: isValid }),
      { wrapper: wrapper(["/?x=bogus"]) },
    );
    expect(result.current[0]).toBe("a");
  });

  it("accepts valid values when validate is provided", () => {
    const isValid = (v: string): v is "a" | "b" => v === "a" || v === "b";
    const { result } = renderHook(
      () => useUrlState("x", "a", { validate: isValid }),
      { wrapper: wrapper(["/?x=b"]) },
    );
    expect(result.current[0]).toBe("b");
  });

  it("multiple instances do not clobber each other", () => {
    const { result } = renderHook(
      () => ({
        tab: useUrlState<string>("tab", "all"),
        sort: useUrlState<string>("sort", "priority"),
      }),
      { wrapper: wrapper(["/?tab=fix&sort=updated"]) },
    );
    expect(result.current.tab[0]).toBe("fix");
    expect(result.current.sort[0]).toBe("updated");

    // Update one param — the other should be preserved
    act(() => {
      result.current.tab[1]("all");
    });
    expect(result.current.tab[0]).toBe("all");
    expect(result.current.sort[0]).toBe("updated");
  });
});
