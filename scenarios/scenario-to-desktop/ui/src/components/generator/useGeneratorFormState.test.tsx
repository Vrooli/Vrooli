import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useGeneratorFormState } from "./useGeneratorFormState";
import { createHookWrapper } from "../../test-utils/renderWithProviders";
import { useFormStore, usePipelineStore } from "../../store";

vi.mock("../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...actual,
    fetchScenarioDesktopStatus: vi.fn().mockResolvedValue({ scenarios: [] }),
    fetchProxyHints: vi.fn().mockResolvedValue(null),
    fetchBundleManifest: vi.fn().mockResolvedValue(null),
    probeEndpoints: vi.fn(),
  };
});

const props = {
  selectedTemplate: "basic",
  onTemplateChange: vi.fn(),
  scenarioName: "",
  onScenarioNameChange: vi.fn(),
  onOpenSigningTab: vi.fn(),
};

describe("useGeneratorFormState", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useFormStore.getState().resetFormState();
    usePipelineStore.getState().reset();
  });

  it("normalizes the server type when a deployment mode only supports external connections", () => {
    const { wrapper } = createHookWrapper();
    const { result } = renderHook(() => useGeneratorFormState(props), {
      wrapper,
    });

    act(() => {
      result.current.setServerType("node");
      result.current.handleDeploymentChange("cloud-api");
    });

    expect(result.current.deploymentMode).toBe("cloud-api");
    expect(result.current.serverType).toBe("external");
  });

  it("applies a saved desktop connection to generation inputs", () => {
    const { wrapper } = createHookWrapper();
    const { result } = renderHook(() => useGeneratorFormState(props), {
      wrapper,
    });

    act(() => {
      result.current.applySavedConnection({
        deployment_mode: "external-server",
        proxy_url: "https://desktop.example.test",
        auto_manage_vrooli: true,
        vrooli_binary_path: "/opt/vrooli/bin/vrooli",
        bundle_manifest_path: "/tmp/app/manifest.json",
        app_display_name: "Example Desktop",
        app_description: "A generated desktop application",
        icon: "/tmp/app/icon.png",
        server_type: "node",
      });
    });

    expect(result.current.deploymentMode).toBe("external-server");
    expect(result.current.proxyUrl).toBe("https://desktop.example.test");
    expect(result.current.autoManageTier1).toBe(true);
    expect(result.current.vrooliBinaryPath).toBe("/opt/vrooli/bin/vrooli");
    expect(result.current.bundleManifestPath).toBe("/tmp/app/manifest.json");
    expect(result.current.appDisplayName).toBe("Example Desktop");
    expect(result.current.appDescription).toBe(
      "A generated desktop application",
    );
    expect(result.current.iconPath).toBe("/tmp/app/icon.png");
    expect(result.current.serverType).toBe("node");
  });

  it("blocks generation and exposes actionable validation errors before starting a pipeline", async () => {
    const { wrapper } = createHookWrapper();
    const { result } = renderHook(() => useGeneratorFormState(props), {
      wrapper,
    });
    const preventDefault = vi.fn();

    await act(async () => {
      await result.current.handleSubmit({ preventDefault } as never);
    });

    await waitFor(() => {
      expect(result.current.validationErrors.length).toBeGreaterThan(0);
    });
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(usePipelineStore.getState().pipelineId).toBeNull();
  });
});
