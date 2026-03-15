import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SandboxList } from "./SandboxList";
import type { Sandbox } from "../lib/api";

function makeSandbox(overrides: Partial<Sandbox> = {}): Sandbox {
  return {
    id: "00000000-0000-0000-0000-000000000001",
    name: "",
    scopePath: "/home/user/Vrooli/scenarios/web-console",
    reservedPath: "/home/user/Vrooli/scenarios/web-console",
    reservedPaths: [],
    noLock: false,
    projectRoot: "/home/user/Vrooli",
    owner: "agent-abc",
    ownerType: "agent",
    status: "active",
    errorMessage: "",
    createdAt: new Date().toISOString(),
    lastUsedAt: new Date().toISOString(),
    sizeBytes: 1024,
    fileCount: 5,
    driver: "overlayfs",
    driverVersion: "1.0",
    lowerDir: "",
    upperDir: "",
    workDir: "",
    mergedDir: "",
    activePids: [],
    sessionCount: 0,
    tags: [],
    metadata: {},
    updatedAt: new Date().toISOString(),
    version: 1,
    ...overrides,
  } as Sandbox;
}

describe("SandboxList", () => {
  const defaultProps = {
    onSelect: vi.fn(),
    isLoading: false,
  };

  describe("R2: Consolidated mount health warnings", () => {
    it("shows a banner when 2+ sandboxes share the same mount warning", () => {
      const mountHealth = { healthy: false, verified: true, error: "mount failed", hint: "Stop and restart" };
      const sandboxes = [
        makeSandbox({ id: "a1", mountHealth }),
        makeSandbox({ id: "a2", mountHealth }),
        makeSandbox({ id: "a3", mountHealth }),
      ];

      render(<SandboxList sandboxes={sandboxes} {...defaultProps} />);

      // Should show 1 banner
      const banners = screen.getAllByTestId("mount-warning-banner");
      expect(banners).toHaveLength(1);
      expect(banners[0]).toHaveTextContent("Stop and restart");
      expect(banners[0]).toHaveTextContent("Affects 3 sandboxes");

      // Should NOT show inline warnings
      expect(screen.queryAllByTestId("mount-warning-inline")).toHaveLength(0);
    });

    it("shows inline warnings when each sandbox has a unique error", () => {
      const sandboxes = [
        makeSandbox({ id: "a1", mountHealth: { healthy: false, verified: true, error: "error A", hint: "" } }),
        makeSandbox({ id: "a2", mountHealth: { healthy: false, verified: true, error: "error B", hint: "" } }),
        makeSandbox({ id: "a3", mountHealth: { healthy: false, verified: true, error: "error C", hint: "" } }),
      ];

      render(<SandboxList sandboxes={sandboxes} {...defaultProps} />);

      // No banners
      expect(screen.queryAllByTestId("mount-warning-banner")).toHaveLength(0);
      // 3 inline warnings
      expect(screen.getAllByTestId("mount-warning-inline")).toHaveLength(3);
    });
  });

  describe("R7: Selected item visibility", () => {
    it("applies strong selection styles to the selected sandbox", () => {
      const sandboxes = [makeSandbox({ id: "s1" })];
      render(<SandboxList sandboxes={sandboxes} selectedId="s1" {...defaultProps} />);

      const item = screen.getByTestId("sandbox-item");
      expect(item.className).toContain("bg-emerald-950/40");
      expect(item.className).toContain("border-l-4");
      expect(item.className).toContain("border-l-emerald-500");
    });

    it("applies transparent border to non-selected items", () => {
      const sandboxes = [makeSandbox({ id: "s1" })];
      render(<SandboxList sandboxes={sandboxes} selectedId="other" {...defaultProps} />);

      const item = screen.getByTestId("sandbox-item");
      expect(item.className).toContain("border-l-transparent");
      expect(item.className).not.toContain("bg-emerald-950/40");
    });
  });

  describe("R10: Card separators", () => {
    it("renders list with divide-y separator classes", () => {
      const sandboxes = [
        makeSandbox({ id: "s1" }),
        makeSandbox({ id: "s2" }),
      ];
      render(<SandboxList sandboxes={sandboxes} {...defaultProps} />);

      const list = screen.getByRole("list");
      expect(list.className).toContain("divide-y");
    });
  });

  describe("R6: Human-friendly labels", () => {
    it("shows derived name from scopePath when no name is set", () => {
      const sandboxes = [
        makeSandbox({
          id: "407578d1-ee26-475a-9ac9-3745b92d5dc3",
          name: "",
          scopePath: "/home/user/Vrooli/scenarios/agent-manager/api",
        }),
      ];
      render(<SandboxList sandboxes={sandboxes} {...defaultProps} />);

      expect(screen.getByText("agent-manager/api")).toBeInTheDocument();
    });

    it("shows sandbox name when provided", () => {
      const sandboxes = [
        makeSandbox({ name: "my-sandbox" }),
      ];
      render(<SandboxList sandboxes={sandboxes} {...defaultProps} />);

      expect(screen.getByText("my-sandbox")).toBeInTheDocument();
    });
  });
});
