/**
 * Tests for PrerequisitesPanel component.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { PrerequisitesPanel } from "./PrerequisitesPanel";
import type { ToolDetectionResult } from "../../lib/api";

describe("PrerequisitesPanel", () => {
  it("renders empty state when no tools provided", () => {
    render(<PrerequisitesPanel tools={[]} />);
    expect(screen.getByText(/Signing tools missing/i)).toBeInTheDocument();
    expect(screen.getByText("Windows")).toBeInTheDocument();
    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.getByText("Linux")).toBeInTheDocument();
  });

  it("shows platform selector in empty state", () => {
    render(<PrerequisitesPanel tools={[]} />);
    const windowsButton = screen.getByRole("button", { name: "Windows" });
    const macosButton = screen.getByRole("button", { name: "macOS" });
    const linuxButton = screen.getByRole("button", { name: "Linux" });

    expect(windowsButton).toBeInTheDocument();
    expect(macosButton).toBeInTheDocument();
    expect(linuxButton).toBeInTheDocument();
  });

  it("switches platform instructions when platform button clicked", () => {
    render(<PrerequisitesPanel tools={[]} />);

    // Click macOS button
    const macosButton = screen.getByRole("button", { name: "macOS" });
    fireEvent.click(macosButton);

    expect(screen.getByText(/macOS setup/i)).toBeInTheDocument();
    expect(screen.getByText(/xcode-select/i)).toBeInTheDocument();
  });

  it("shows refresh button when onRefresh provided in empty state", () => {
    const onRefresh = vi.fn();
    render(<PrerequisitesPanel tools={[]} onRefresh={onRefresh} />);

    const refreshButton = screen.getByRole("button", { name: /re-scan/i });
    expect(refreshButton).toBeInTheDocument();
  });

  it("calls onRefresh when refresh button clicked", () => {
    const onRefresh = vi.fn();
    render(<PrerequisitesPanel tools={[]} onRefresh={onRefresh} />);

    const refreshButton = screen.getByRole("button", { name: /re-scan/i });
    fireEvent.click(refreshButton);

    expect(onRefresh).toHaveBeenCalled();
  });

  it("disables refresh button and shows loading state when refreshing", () => {
    const onRefresh = vi.fn();
    render(<PrerequisitesPanel tools={[]} onRefresh={onRefresh} refreshing={true} />);

    const refreshButton = screen.getByRole("button", { name: /re-scanning/i });
    expect(refreshButton).toBeDisabled();
  });

  it("renders tools grouped by platform when tools provided", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "signtool", platform: "windows", installed: true, version: "10.0" },
      { tool: "codesign", platform: "macos", installed: true },
      { tool: "gpg", platform: "linux", installed: false },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("Windows")).toBeInTheDocument();
    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.getByText("Linux")).toBeInTheDocument();
  });

  it("shows installed status for installed tools", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "codesign", platform: "macos", installed: true, version: "1.0" },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("codesign")).toBeInTheDocument();
    expect(screen.getByText("v1.0")).toBeInTheDocument();
  });

  it("shows tool path when provided", () => {
    const tools: ToolDetectionResult[] = [
      {
        tool: "gpg",
        platform: "linux",
        installed: true,
        path: "/usr/bin/gpg",
      },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("/usr/bin/gpg")).toBeInTheDocument();
  });

  it("shows error message for tools with errors", () => {
    const tools: ToolDetectionResult[] = [
      {
        tool: "signtool",
        platform: "windows",
        installed: false,
        error: "Permission denied",
      },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("Permission denied")).toBeInTheDocument();
  });

  it("shows remediation message for tools with remediation", () => {
    const tools: ToolDetectionResult[] = [
      {
        tool: "osslsigncode",
        platform: "linux",
        installed: false,
        remediation: "Install via apt: apt install osslsigncode",
      },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("Install via apt: apt install osslsigncode")).toBeInTheDocument();
  });

  it("shows install hints for missing tools", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "codesign", platform: "macos", installed: false },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    // There are multiple mentions of xcode-select, so check for at least one
    const installHints = screen.getAllByText(/xcode-select --install/i);
    expect(installHints.length).toBeGreaterThan(0);
  });

  it("renders legend showing status icons", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "gpg", platform: "linux", installed: true },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("Installed")).toBeInTheDocument();
    expect(screen.getByText("Not found")).toBeInTheDocument();
    expect(screen.getByText("Issue detected")).toBeInTheDocument();
  });

  it("renders tool descriptions", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "gpg", platform: "linux", installed: true },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText(/GNU Privacy Guard/i)).toBeInTheDocument();
  });

  it("handles tools with all platform", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "custom", platform: "all", installed: true },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    expect(screen.getByText("All Platforms")).toBeInTheDocument();
  });

  it("maintains platform order: windows, macos, linux, all", () => {
    const tools: ToolDetectionResult[] = [
      { tool: "gpg", platform: "linux", installed: true },
      { tool: "custom", platform: "all", installed: true },
      { tool: "codesign", platform: "macos", installed: true },
      { tool: "signtool", platform: "windows", installed: true },
    ];
    render(<PrerequisitesPanel tools={tools} />);

    const platformHeaders = screen.getAllByRole("heading", { level: 4 });
    expect(platformHeaders[0]).toHaveTextContent("Windows");
    expect(platformHeaders[1]).toHaveTextContent("macOS");
    expect(platformHeaders[2]).toHaveTextContent("Linux");
    expect(platformHeaders[3]).toHaveTextContent("All Platforms");
  });
});
