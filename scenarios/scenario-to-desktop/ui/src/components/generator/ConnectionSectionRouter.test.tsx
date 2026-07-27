import { render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { ConnectionSectionRouter } from "./ConnectionSectionRouter";

vi.mock("../runtime", () => ({
  ExternalServerSection: ({ scenarioName }: { scenarioName: string }) => (
    <div>External for {scenarioName}</div>
  ),
  EmbeddedServerSection: ({ serverPort }: { serverPort: number }) => (
    <div>Embedded on {serverPort}</div>
  ),
}));

function props(kind: "bundled-runtime" | "remote-server" | "local-embedded") {
  return {
    connectionDecision: { kind } as never,
    scenarioName: "canvas-lab",
    proxyUrl: "https://remote.example/proxy/",
    onProxyUrlChange: vi.fn(),
    proxyHints: null,
    connectionTester: { isPending: false, mutate: vi.fn() },
    connectionResult: null,
    connectionError: null,
    autoManageTier1: false,
    onAutoManageTier1Change: vi.fn(),
    vrooliBinaryPath: "vrooli",
    onVrooliBinaryPathChange: vi.fn(),
    serverPort: 3900,
    onServerPortChange: vi.fn(),
    localServerPath: "api/main.js",
    onLocalServerPathChange: vi.fn(),
    localApiEndpoint: "http://127.0.0.1:3900",
    onLocalApiEndpointChange: vi.fn(),
  };
}

describe("ConnectionSectionRouter", () => {
  it("delegates remote deployments to the secure external-server form", () => {
    render(<ConnectionSectionRouter {...props("remote-server")} />);
    expect(screen.getByText("External for canvas-lab")).toBeInTheDocument();
  });

  it("delegates local embedded deployments to the local-runtime form", () => {
    render(<ConnectionSectionRouter {...props("local-embedded")} />);
    expect(screen.getByText("Embedded on 3900")).toBeInTheDocument();
  });

  it("does not render a duplicate connection form for bundled deployments", () => {
    const { container } = render(
      <ConnectionSectionRouter {...props("bundled-runtime")} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
