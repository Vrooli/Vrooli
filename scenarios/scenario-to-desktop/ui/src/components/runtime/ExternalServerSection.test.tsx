import { fireEvent, render, screen } from "@/test-utils";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import { ExternalServerSection } from "./ExternalServerSection";

function renderSection(overrides: Partial<ComponentProps<typeof ExternalServerSection>> = {}) {
  const props = {
    proxyUrl: "https://monitor.example/apps/canvas-lab/proxy/",
    onProxyUrlChange: vi.fn(),
    scenarioName: "canvas-lab",
    connectionTester: { isPending: false, mutate: vi.fn() },
    connectionResult: null,
    connectionError: null,
    autoManageTier1: false,
    onAutoManageTier1Change: vi.fn(),
    vrooliBinaryPath: "vrooli",
    onVrooliBinaryPathChange: vi.fn(),
    ...overrides,
  };
  render(<ExternalServerSection {...props} />);
  return props;
}

describe("ExternalServerSection", () => {
  it("edits a tunnel URL, chooses detected URLs, and requests a connectivity test", () => {
    const props = renderSection({
      proxyHints: {
        hints: [{
          url: "https://remote.example/apps/canvas-lab/proxy/",
          message: "Cloudflare tunnel",
          source: "app-monitor",
        }],
      } as never,
    });

    fireEvent.change(screen.getByLabelText("Proxy URL"), { target: { value: "https://new.example/proxy/" } });
    expect(props.onProxyUrlChange).toHaveBeenCalledWith("https://new.example/proxy/");
    fireEvent.click(screen.getByRole("button", { name: /remote.example/ }));
    expect(props.onProxyUrlChange).toHaveBeenCalledWith("https://remote.example/apps/canvas-lab/proxy/");
    fireEvent.click(screen.getByRole("button", { name: "Test connection" }));
    expect(props.connectionTester.mutate).toHaveBeenCalledOnce();
  });

  it("reports connection evidence, errors, pending state, and local CLI management", () => {
    const props = renderSection({
      connectionTester: { isPending: true, mutate: vi.fn() },
      connectionResult: {
        server: { status: "ok" },
        api: { status: "error", message: "API denied" },
      } as never,
      connectionError: "Tunnel authentication failed",
      autoManageTier1: true,
    });

    expect(screen.getByRole("button", { name: "Testing..." })).toBeDisabled();
    expect(screen.getByText("Connectivity snapshot")).toBeInTheDocument();
    expect(screen.getByText(/UI URL: reachable/)).toBeInTheDocument();
    expect(screen.getByText(/API URL: API denied/)).toBeInTheDocument();
    expect(screen.getByText("Tunnel authentication failed")).toBeInTheDocument();
    const autoManage = screen.getByLabelText(/Automatically run the scenario/);
    expect(autoManage).toBeChecked();
    fireEvent.click(autoManage);
    expect(props.onAutoManageTier1Change).toHaveBeenCalledWith(false);
    fireEvent.change(screen.getByLabelText("vrooli CLI path"), { target: { value: "/usr/local/bin/vrooli" } });
    expect(props.onVrooliBinaryPathChange).toHaveBeenCalledWith("/usr/local/bin/vrooli");
  });

  it("disables unsafe actions until a URL is provided and when local management is off", () => {
    renderSection({ proxyUrl: "", autoManageTier1: false });
    expect(screen.getByRole("button", { name: "Test connection" })).toBeDisabled();
    expect(screen.getByLabelText("vrooli CLI path")).toBeDisabled();
  });
});
