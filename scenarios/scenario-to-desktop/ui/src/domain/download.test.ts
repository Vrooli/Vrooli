import { describe, it, expect } from "vitest";
import {
  isValidPlatform,
  parsePlatform,
  getPlatformDisplayInfo,
  getPlatformIcon,
  getPlatformName,
  groupArtifactsByPlatform,
  getSortedPlatformGroups,
  formatBytes,
  computeTotalArtifactSize,
  buildDownloadPath,
  hasDownloadableArtifacts,
  getAvailablePlatforms,
  VALID_PLATFORMS,
  type DesktopBuildArtifact
} from "./download";

describe("download domain", () => {
  describe("isValidPlatform", () => {
    it("returns true for valid platforms", () => {
      expect(isValidPlatform("win")).toBe(true);
      expect(isValidPlatform("mac")).toBe(true);
      expect(isValidPlatform("linux")).toBe(true);
    });

    it("returns false for invalid platforms", () => {
      expect(isValidPlatform("windows")).toBe(false);
      expect(isValidPlatform("macos")).toBe(false);
      expect(isValidPlatform("ubuntu")).toBe(false);
      expect(isValidPlatform("")).toBe(false);
    });
  });

  describe("parsePlatform", () => {
    it("returns the platform for valid strings", () => {
      expect(parsePlatform("win")).toBe("win");
      expect(parsePlatform("mac")).toBe("mac");
      expect(parsePlatform("linux")).toBe("linux");
    });

    it("returns undefined for invalid strings", () => {
      expect(parsePlatform("invalid")).toBeUndefined();
      expect(parsePlatform(undefined)).toBeUndefined();
      expect(parsePlatform("")).toBeUndefined();
    });
  });

  describe("getPlatformDisplayInfo", () => {
    it("returns correct info for known platforms", () => {
      expect(getPlatformDisplayInfo("win")).toEqual({
        icon: "🪟",
        name: "Windows",
        shortName: "Win"
      });
      expect(getPlatformDisplayInfo("mac")).toEqual({
        icon: "🍎",
        name: "macOS",
        shortName: "Mac"
      });
      expect(getPlatformDisplayInfo("linux")).toEqual({
        icon: "🐧",
        name: "Linux",
        shortName: "Linux"
      });
    });

    it("returns fallback for unknown platforms", () => {
      expect(getPlatformDisplayInfo("unknown")).toEqual({
        icon: "📦",
        name: "Unknown",
        shortName: "?"
      });
    });
  });

  describe("getPlatformIcon", () => {
    it("returns icons for known platforms", () => {
      expect(getPlatformIcon("win")).toBe("🪟");
      expect(getPlatformIcon("mac")).toBe("🍎");
      expect(getPlatformIcon("linux")).toBe("🐧");
    });
  });

  describe("getPlatformName", () => {
    it("returns names for known platforms", () => {
      expect(getPlatformName("win")).toBe("Windows");
      expect(getPlatformName("mac")).toBe("macOS");
      expect(getPlatformName("linux")).toBe("Linux");
    });
  });

  describe("formatBytes", () => {
    it("formats 0 bytes", () => {
      expect(formatBytes(0)).toBe("0 Bytes");
      expect(formatBytes(undefined)).toBe("0 Bytes");
    });

    it("formats bytes", () => {
      expect(formatBytes(500)).toBe("500 Bytes");
    });

    it("formats kilobytes", () => {
      expect(formatBytes(1024)).toBe("1 KB");
      expect(formatBytes(1536)).toBe("1.5 KB");
    });

    it("formats megabytes", () => {
      expect(formatBytes(1048576)).toBe("1 MB");
      expect(formatBytes(1572864)).toBe("1.5 MB");
    });

    it("formats gigabytes", () => {
      expect(formatBytes(1073741824)).toBe("1 GB");
    });
  });

  describe("groupArtifactsByPlatform", () => {
    const testArtifacts: DesktopBuildArtifact[] = [
      { platform: "win", file_name: "app.exe", size_bytes: 1000 },
      { platform: "win", file_name: "app.msi", size_bytes: 2000 },
      { platform: "mac", file_name: "app.dmg", size_bytes: 3000 },
      { platform: "linux", file_name: "app.AppImage", size_bytes: 4000 }
    ];

    it("groups artifacts by platform", () => {
      const groups = groupArtifactsByPlatform(testArtifacts);

      expect(groups.get("win")?.artifacts).toHaveLength(2);
      expect(groups.get("mac")?.artifacts).toHaveLength(1);
      expect(groups.get("linux")?.artifacts).toHaveLength(1);
    });

    it("computes total size per platform", () => {
      const groups = groupArtifactsByPlatform(testArtifacts);

      expect(groups.get("win")?.totalSizeBytes).toBe(3000);
      expect(groups.get("mac")?.totalSizeBytes).toBe(3000);
      expect(groups.get("linux")?.totalSizeBytes).toBe(4000);
    });

    it("handles empty or undefined artifacts", () => {
      expect(groupArtifactsByPlatform(undefined).size).toBe(0);
      expect(groupArtifactsByPlatform([]).size).toBe(0);
    });

    it("handles artifacts with unknown platform", () => {
      const artifacts = [{ platform: undefined, file_name: "test.zip" }];
      const groups = groupArtifactsByPlatform(artifacts);

      expect(groups.has("unknown")).toBe(true);
    });
  });

  describe("getSortedPlatformGroups", () => {
    const testArtifacts: DesktopBuildArtifact[] = [
      { platform: "linux", file_name: "app.AppImage" },
      { platform: "win", file_name: "app.exe" },
      { platform: "mac", file_name: "app.dmg" }
    ];

    it("returns groups in platform order (win, mac, linux)", () => {
      const groups = getSortedPlatformGroups(testArtifacts);

      expect(groups.map((g) => g.platform)).toEqual(["win", "mac", "linux"]);
    });

    it("returns empty array for no artifacts", () => {
      expect(getSortedPlatformGroups(undefined)).toEqual([]);
      expect(getSortedPlatformGroups([])).toEqual([]);
    });
  });

  describe("computeTotalArtifactSize", () => {
    it("sums artifact sizes", () => {
      const artifacts: DesktopBuildArtifact[] = [
        { file_name: "a", size_bytes: 100 },
        { file_name: "b", size_bytes: 200 },
        { file_name: "c", size_bytes: 300 }
      ];

      expect(computeTotalArtifactSize(artifacts)).toBe(600);
    });

    it("handles missing size_bytes", () => {
      const artifacts: DesktopBuildArtifact[] = [
        { file_name: "a", size_bytes: 100 },
        { file_name: "b" },
        { file_name: "c", size_bytes: 300 }
      ];

      expect(computeTotalArtifactSize(artifacts)).toBe(400);
    });

    it("returns 0 for empty or undefined", () => {
      expect(computeTotalArtifactSize(undefined)).toBe(0);
      expect(computeTotalArtifactSize([])).toBe(0);
    });
  });

  describe("buildDownloadPath", () => {
    it("builds correct path for valid inputs", () => {
      expect(buildDownloadPath({ scenarioName: "my-app", platform: "win" })).toBe(
        "/desktop/download/my-app/win"
      );
      expect(buildDownloadPath({ scenarioName: "test", platform: "mac" })).toBe(
        "/desktop/download/test/mac"
      );
    });

    it("encodes special characters in scenario name", () => {
      expect(buildDownloadPath({ scenarioName: "my app", platform: "linux" })).toBe(
        "/desktop/download/my%20app/linux"
      );
    });

    it("throws for empty scenario name", () => {
      expect(() => buildDownloadPath({ scenarioName: "", platform: "win" })).toThrow(
        "scenarioName is required"
      );
    });

    it("throws for invalid platform", () => {
      expect(() =>
        buildDownloadPath({ scenarioName: "test", platform: "invalid" as never })
      ).toThrow("Invalid platform");
    });
  });

  describe("hasDownloadableArtifacts", () => {
    const artifacts: DesktopBuildArtifact[] = [
      { platform: "win", file_name: "app.exe" },
      { platform: "mac", file_name: "app.dmg" }
    ];

    it("returns true when platform has artifacts", () => {
      expect(hasDownloadableArtifacts(artifacts, "win")).toBe(true);
      expect(hasDownloadableArtifacts(artifacts, "mac")).toBe(true);
    });

    it("returns false when platform has no artifacts", () => {
      expect(hasDownloadableArtifacts(artifacts, "linux")).toBe(false);
    });

    it("returns false for empty or undefined", () => {
      expect(hasDownloadableArtifacts(undefined, "win")).toBe(false);
      expect(hasDownloadableArtifacts([], "win")).toBe(false);
    });
  });

  describe("getAvailablePlatforms", () => {
    it("returns platforms in standard order", () => {
      const artifacts: DesktopBuildArtifact[] = [
        { platform: "linux", file_name: "app.AppImage" },
        { platform: "win", file_name: "app.exe" }
      ];

      expect(getAvailablePlatforms(artifacts)).toEqual(["win", "linux"]);
    });

    it("returns empty for no artifacts", () => {
      expect(getAvailablePlatforms(undefined)).toEqual([]);
      expect(getAvailablePlatforms([])).toEqual([]);
    });

    it("deduplicates platforms", () => {
      const artifacts: DesktopBuildArtifact[] = [
        { platform: "win", file_name: "app.exe" },
        { platform: "win", file_name: "app.msi" }
      ];

      expect(getAvailablePlatforms(artifacts)).toEqual(["win"]);
    });
  });

  describe("VALID_PLATFORMS constant", () => {
    it("contains all supported platforms", () => {
      expect(VALID_PLATFORMS).toContain("win");
      expect(VALID_PLATFORMS).toContain("mac");
      expect(VALID_PLATFORMS).toContain("linux");
      expect(VALID_PLATFORMS).toHaveLength(3);
    });
  });
});
