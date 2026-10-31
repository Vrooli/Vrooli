import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { ComponentProps } from "react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const api = vi.hoisted(() => ({
  listBundles: vi.fn(), cleanupBundles: vi.fn(), deleteBundle: vi.fn(), listVPSBundles: vi.fn(), deleteVPSBundle: vi.fn(),
}));
vi.mock("./lib/api", () => api);

import { BuildStatusPanel, StepBuild } from "./components/wizard/StepBuild";

const bundle = { filename: "demo.tar.gz", path: "/tmp/demo.tar.gz", sha256: "sha-demo", size_bytes: 2048, created_at: new Date().toISOString(), scenario_id: "demo" };
const vpsBundle = { filename: "remote.tar.gz", sha256: "sha-remote", size_bytes: 1024, created_at: new Date().toISOString() };
const deployment = {
  parsedManifest: { ok: true, value: { scenario: { id: "demo" }, target: { vps: { host: "vps.example", key_path: "/tmp/key", port: 2222, user: "deploy", workdir: "/opt/app" } } } },
  bundleArtifact: null, bundleError: null, isBuildingBundle: false, build: vi.fn(), setBundleArtifact: vi.fn(),
};

describe("bundle build surfaces", () => {
  beforeEach(() => {
    api.listBundles.mockResolvedValue({ bundles: [bundle] });
    api.cleanupBundles.mockResolvedValue({ message: "cleaned" });
    api.deleteBundle.mockResolvedValue({ ok: true });
    api.listVPSBundles.mockResolvedValue({ ok: true, bundles: [vpsBundle] });
    api.deleteVPSBundle.mockResolvedValue({ ok: true });
  });

  it("renders interactive and readonly artifact states", () => {
    const build = vi.fn(); const clear = vi.fn();
    const { rerender } = renderWithProviders(<BuildStatusPanel isBuilding={false} bundleArtifact={null} canBuild onBuild={build} onClearBundle={clear} />);
    expect(screen.getByText(/Build a new bundle/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Build New Bundle" }));
    expect(build).toHaveBeenCalled();
    rerender(<BuildStatusPanel isBuilding bundleArtifact={null} />);
    expect(screen.getByText(/Building mini/)).toBeInTheDocument();
    rerender(<BuildStatusPanel isBuilding={false} bundleArtifact={null} bundleError="bad bundle" />);
    expect(screen.getByText("bad bundle")).toBeInTheDocument();
    rerender(<BuildStatusPanel isBuilding={false} bundleArtifact={{ path: "/x", sha256: "s", size_bytes: 0 }} onClearBundle={clear} />);
    expect(screen.getByText("Bundle Artifact")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Change Build" }));
    expect(clear).toHaveBeenCalled();
    rerender(<BuildStatusPanel mode="readonly" isBuilding={false} bundleArtifact={null} bundlePathFallback="/fallback" bundleShaFallback="fallback-sha" bundleSizeBytesFallback={1024} />);
    expect(screen.getByText("Bundle is ready for deployment.")).toBeInTheDocument();
  });

  it("lists, selects, deletes, cleans, and refreshes local and VPS bundles", async () => {
    const setArtifact = vi.fn();
    renderWithProviders(<StepBuild deployment={{ ...deployment, setBundleArtifact: setArtifact } as unknown as ComponentProps<typeof StepBuild>["deployment"]} />);
    await waitFor(() => expect(screen.getByText("demo.tar.gz")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Use This Build" }));
    expect(setArtifact).toHaveBeenCalled();
  });
});
