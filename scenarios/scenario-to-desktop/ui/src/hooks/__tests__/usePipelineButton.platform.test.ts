/**
 * Tests for usePlatformSelection and useWineCheck hooks.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import {
  usePlatformSelection,
  useWineCheck,
} from "../usePipelineButton";
import type { WineCheckResponse } from "../../lib/api";
import {
  mockCheckWineStatus,
  createWrapper,
  localStorageMock,
} from "./usePipelineButton.testUtils";

beforeEach(() => {
  vi.clearAllMocks();
  localStorageMock.clear();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("usePlatformSelection", () => {
  it("returns default platforms when no stored value", () => {
    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac", "linux"]);
  });

  it("uses custom default platforms", () => {
    const { result } = renderHook(
      () =>
        usePlatformSelection({
          storageKey: "test-platforms",
          defaultPlatforms: ["win"],
        }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win"]);
  });

  it("loads platforms from localStorage", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify(["win", "mac"]));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac"]);
  });

  it("falls back to defaults on invalid localStorage data", () => {
    localStorageMock.getItem.mockReturnValue("invalid json");

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac", "linux"]);
  });

  it("falls back to defaults when localStorage contains non-array", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify({ not: "array" }));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac", "linux"]);
  });

  it("saves to localStorage when platforms change", async () => {
    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setSelectedPlatforms(["win"]);
    });

    await waitFor(() => {
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        "test-platforms",
        JSON.stringify(["win"])
      );
    });
  });

  it("togglePlatform adds platform when not selected", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify(["win"]));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.togglePlatform("mac");
    });

    expect(result.current.selectedPlatforms).toEqual(["win", "mac"]);
  });

  it("togglePlatform removes platform when selected", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify(["win", "mac"]));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.togglePlatform("mac");
    });

    expect(result.current.selectedPlatforms).toEqual(["win"]);
  });
});

describe("useWineCheck", () => {
  it("fetches wine check status", async () => {
    const mockWineCheck: WineCheckResponse = {
      installed: true,
      platform: "linux",
      version: "wine-9.0",
    };
    mockCheckWineStatus.mockResolvedValue(mockWineCheck);

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.wineCheck).toEqual(mockWineCheck);
    });
  });

  it("showWineDialog starts as false", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    expect(result.current.showWineDialog).toBe(false);
  });

  it("setShowWineDialog updates state", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.setShowWineDialog(true);
    });

    expect(result.current.showWineDialog).toBe(true);
  });

  it("pendingPlatforms starts as empty", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    expect(result.current.pendingPlatforms).toEqual([]);
  });

  it("setPendingPlatforms updates state", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.setPendingPlatforms(["win", "mac"]);
    });

    expect(result.current.pendingPlatforms).toEqual(["win", "mac"]);
  });

  describe("needsWineForPlatforms", () => {
    it("returns true when win platform on linux without wine", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: false,
        platform: "linux",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["win"])).toBe(true);
      expect(result.current.needsWineForPlatforms(["win", "mac"])).toBe(true);
    });

    it("returns false when wine is installed", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: true,
        platform: "linux",
        version: "wine-9.0",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["win"])).toBe(false);
    });

    it("returns false when not on linux", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: false,
        platform: "darwin",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["win"])).toBe(false);
    });

    it("returns false when win not in platforms", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: false,
        platform: "linux",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["mac", "linux"])).toBe(false);
    });
  });

  it("handleWineInstallComplete closes dialog and invalidates query", async () => {
    mockCheckWineStatus
      .mockResolvedValueOnce({ installed: false, platform: "linux" })
      .mockResolvedValueOnce({ installed: true, platform: "linux", version: "wine-9.0" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.wineCheck).toBeDefined();
    });

    act(() => {
      result.current.setShowWineDialog(true);
    });

    expect(result.current.showWineDialog).toBe(true);

    act(() => {
      result.current.handleWineInstallComplete();
    });

    expect(result.current.showWineDialog).toBe(false);
  });
});
