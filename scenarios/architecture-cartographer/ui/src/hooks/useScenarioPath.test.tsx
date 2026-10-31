import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { ReactNode } from "react";

import { encodeScenarioPath, useScenarioPath } from "./useScenarioPath";

const wrap = (encoded: string | null) => {
  const path = encoded === null ? "/" : `/targets/${encoded}`;
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter
      initialEntries={[path]}
      future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
    >
      <Routes>
        <Route path="/" element={children} />
        <Route path="/targets/:encodedPath" element={children} />
      </Routes>
    </MemoryRouter>
  );
  return Wrapper;
};

describe("useScenarioPath", () => {
  it("returns null when the param is absent", () => {
    const { result } = renderHook(() => useScenarioPath(), { wrapper: wrap(null) });
    expect(result.current).toBeNull();
  });

  it("decodes a percent-encoded path", () => {
    const encoded = encodeScenarioPath("scenarios/architecture-cartographer");
    const { result } = renderHook(() => useScenarioPath(), { wrapper: wrap(encoded) });
    expect(result.current).toBe("scenarios/architecture-cartographer");
  });

});

describe("encodeScenarioPath", () => {
  it("escapes slashes so the path round-trips through a URL segment", () => {
    expect(encodeScenarioPath("a/b c")).toBe("a%2Fb%20c");
  });
});
