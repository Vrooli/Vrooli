/**
 * Tests for useUrlState hook.
 * Tests URL-based state synchronization and browser navigation support.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import {
  useUrlState,
  parseSearchParams,
  type ViewMode,
} from "./useUrlState";

// Store original values
const originalLocation = window.location;
const _originalHistory = window.history;

// Create a mock URL that behaves like window.location
function createMockLocation(url: string) {
  const urlObj = new URL(url);
  return {
    ...originalLocation,
    href: urlObj.href,
    search: urlObj.search,
    origin: urlObj.origin,
    pathname: urlObj.pathname,
    hash: urlObj.hash,
    toString: () => urlObj.href,
  };
}

beforeEach(() => {
  vi.clearAllMocks();

  // Reset to default URL
  Object.defineProperty(window, "location", {
    value: createMockLocation("http://localhost:5173/"),
    writable: true,
    configurable: true,
  });

  // Mock history.replaceState
  vi.spyOn(window.history, "replaceState").mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("parseSearchParams", () => {
  it("returns empty object when no params", () => {
    Object.defineProperty(window, "location", {
      value: createMockLocation("http://localhost:5173/"),
      writable: true,
      configurable: true,
    });

    const result = parseSearchParams();
    expect(result).toEqual({});
  });

  it("parses view param", () => {
    Object.defineProperty(window, "location", {
      value: createMockLocation("http://localhost:5173/?view=generator"),
      writable: true,
      configurable: true,
    });

    const result = parseSearchParams();
    expect(result.view).toBe("generator");
  });

  it("parses scenario param", () => {
    Object.defineProperty(window, "location", {
      value: createMockLocation("http://localhost:5173/?scenario=my-scenario"),
      writable: true,
      configurable: true,
    });

    const result = parseSearchParams();
    expect(result.scenario).toBe("my-scenario");
  });

  it("parses doc param", () => {
    Object.defineProperty(window, "location", {
      value: createMockLocation("http://localhost:5173/?doc=/docs/getting-started"),
      writable: true,
      configurable: true,
    });

    const result = parseSearchParams();
    expect(result.doc).toBe("/docs/getting-started");
  });

  it("parses all params together", () => {
    Object.defineProperty(window, "location", {
      value: createMockLocation(
        "http://localhost:5173/?view=docs&scenario=test-scenario&doc=/docs/intro"
      ),
      writable: true,
      configurable: true,
    });

    const result = parseSearchParams();
    expect(result).toEqual({
      view: "docs",
      scenario: "test-scenario",
      doc: "/docs/intro",
    });
  });

  it("returns undefined view for empty view param", () => {
    Object.defineProperty(window, "location", {
      value: createMockLocation("http://localhost:5173/?view="),
      writable: true,
      configurable: true,
    });

    const result = parseSearchParams();
    expect(result.view).toBeUndefined();
  });
});

describe("useUrlState", () => {
  describe("initial state", () => {
    it("uses default view when no URL param", () => {
      const { result } = renderHook(() => useUrlState());

      expect(result.current.viewMode).toBe("inventory");
    });

    it("uses custom default view", () => {
      const { result } = renderHook(() =>
        useUrlState({ defaultView: "generator" })
      );

      expect(result.current.viewMode).toBe("generator");
    });

    it("uses view from URL when present", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?view=docs"),
        writable: true,
        configurable: true,
      });

      const { result } = renderHook(() => useUrlState());

      expect(result.current.viewMode).toBe("docs");
    });

    it("uses scenario from URL when present", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?scenario=my-scenario"),
        writable: true,
        configurable: true,
      });

      const { result } = renderHook(() => useUrlState());

      expect(result.current.scenarioName).toBe("my-scenario");
    });

    it("starts with empty scenario when not in URL", () => {
      const { result } = renderHook(() => useUrlState());

      expect(result.current.scenarioName).toBe("");
    });

    it("uses doc path from URL when present", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?doc=/docs/intro"),
        writable: true,
        configurable: true,
      });

      const { result } = renderHook(() => useUrlState());

      expect(result.current.docPath).toBe("/docs/intro");
    });

    it("starts with null doc path when not in URL", () => {
      const { result } = renderHook(() => useUrlState());

      expect(result.current.docPath).toBeNull();
    });

    it("provides initial params", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?view=signing&scenario=test"),
        writable: true,
        configurable: true,
      });

      const { result } = renderHook(() => useUrlState());

      expect(result.current.initialParams).toEqual({
        view: "signing",
        scenario: "test",
        doc: undefined,
      });
    });
  });

  describe("setViewMode", () => {
    it("updates view mode state", () => {
      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setViewMode("generator");
      });

      expect(result.current.viewMode).toBe("generator");
    });

    it("calls onViewChange callback", () => {
      const onViewChange = vi.fn();

      const { result } = renderHook(() =>
        useUrlState({ onViewChange })
      );

      act(() => {
        result.current.setViewMode("records");
      });

      expect(onViewChange).toHaveBeenCalledWith("records");
    });

    it("updates URL with new view", () => {
      const replaceSpy = vi.spyOn(window.history, "replaceState");

      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setViewMode("distribution");
      });

      expect(replaceSpy).toHaveBeenCalled();
      const lastCall = replaceSpy.mock.calls[replaceSpy.mock.calls.length - 1];
      expect(lastCall?.[2]).toContain("view=distribution");
    });
  });

  describe("setScenarioName", () => {
    it("updates scenario name state", () => {
      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setScenarioName("my-scenario");
      });

      expect(result.current.scenarioName).toBe("my-scenario");
    });

    it("calls onScenarioChange callback", () => {
      const onScenarioChange = vi.fn();

      const { result } = renderHook(() =>
        useUrlState({ onScenarioChange })
      );

      act(() => {
        result.current.setScenarioName("test-scenario");
      });

      expect(onScenarioChange).toHaveBeenCalledWith("test-scenario");
    });

    it("adds scenario to URL when set", () => {
      const replaceSpy = vi.spyOn(window.history, "replaceState");

      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setScenarioName("new-scenario");
      });

      expect(replaceSpy).toHaveBeenCalled();
      const lastCall = replaceSpy.mock.calls[replaceSpy.mock.calls.length - 1];
      expect(lastCall?.[2]).toContain("scenario=new-scenario");
    });

    it("removes scenario from URL when cleared", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?view=inventory&scenario=old-scenario"),
        writable: true,
        configurable: true,
      });

      const replaceSpy = vi.spyOn(window.history, "replaceState");

      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setScenarioName("");
      });

      expect(replaceSpy).toHaveBeenCalled();
      const lastCall = replaceSpy.mock.calls[replaceSpy.mock.calls.length - 1];
      expect(lastCall?.[2]).not.toContain("scenario=");
    });
  });

  describe("setDocPath", () => {
    it("updates doc path state", () => {
      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setDocPath("/docs/guide");
      });

      expect(result.current.docPath).toBe("/docs/guide");
    });

    it("calls onDocChange callback", () => {
      const onDocChange = vi.fn();

      const { result } = renderHook(() =>
        useUrlState({ onDocChange })
      );

      act(() => {
        result.current.setDocPath("/docs/api");
      });

      expect(onDocChange).toHaveBeenCalledWith("/docs/api");
    });

    it("calls onDocChange with null when clearing", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?doc=/docs/old"),
        writable: true,
        configurable: true,
      });

      const onDocChange = vi.fn();

      const { result } = renderHook(() =>
        useUrlState({ onDocChange })
      );

      act(() => {
        result.current.setDocPath(null);
      });

      expect(onDocChange).toHaveBeenCalledWith(null);
    });

    it("adds doc to URL when set", () => {
      const replaceSpy = vi.spyOn(window.history, "replaceState");

      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setDocPath("/docs/intro");
      });

      expect(replaceSpy).toHaveBeenCalled();
      const lastCall = replaceSpy.mock.calls[replaceSpy.mock.calls.length - 1];
      expect(lastCall?.[2]).toContain("doc=");
    });

    it("removes doc from URL when cleared", () => {
      Object.defineProperty(window, "location", {
        value: createMockLocation("http://localhost:5173/?view=docs&doc=/docs/old"),
        writable: true,
        configurable: true,
      });

      const replaceSpy = vi.spyOn(window.history, "replaceState");

      const { result } = renderHook(() => useUrlState());

      act(() => {
        result.current.setDocPath(null);
      });

      expect(replaceSpy).toHaveBeenCalled();
      const lastCall = replaceSpy.mock.calls[replaceSpy.mock.calls.length - 1];
      expect(lastCall?.[2]).not.toContain("doc=");
    });
  });

  describe("all view modes", () => {
    const viewModes: ViewMode[] = [
      "generator",
      "inventory",
      "docs",
      "records",
      "signing",
      "distribution",
    ];

    viewModes.forEach((mode) => {
      it(`supports ${mode} view mode`, () => {
        const { result } = renderHook(() => useUrlState());

        act(() => {
          result.current.setViewMode(mode);
        });

        expect(result.current.viewMode).toBe(mode);
      });
    });
  });
});
