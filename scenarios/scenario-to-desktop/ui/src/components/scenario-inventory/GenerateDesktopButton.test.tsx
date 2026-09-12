import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GenerateDesktopButton } from "./GenerateDesktopButton";
import {
  DeploymentMode,
  StageName,
  TemplateType,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type { ScenarioDesktopStatus } from "./types";

const { resetMock, runPipelineWithConfigMock, probeEndpointsMock } = vi.hoisted(
  () => ({
    resetMock: vi.fn(),
    runPipelineWithConfigMock: vi.fn(),
    probeEndpointsMock: vi.fn(),
  }),
);

vi.mock("../../lib/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../lib/api")>()),
  probeEndpoints: probeEndpointsMock,
}));

vi.mock("../../hooks", () => ({
  usePipelineMutation: () => ({
    state: { buildId: null, error: null },
    mutation: { isPending: false, isError: false, error: null },
    runPipelineWithConfig: runPipelineWithConfigMock,
    reset: resetMock,
  }),
  usePipelineStatus: () => ({
    pipelineStatus: null,
    isBuilding: false,
    isComplete: false,
    isFailed: false,
  }),
}));

function scenario(
  overrides: Partial<ScenarioDesktopStatus> = {},
): ScenarioDesktopStatus {
  return {
    name: "secrets-manager",
    has_desktop: false,
    ...overrides,
  };
}

describe("GenerateDesktopButton", () => {
  beforeEach(() => {
    resetMock.mockReset();
    runPipelineWithConfigMock.mockReset();
    probeEndpointsMock.mockReset();
  });

  it("generates a bundled desktop wrapper only with the exported bundle manifest", async () => {
    render(
      <GenerateDesktopButton
        scenario={scenario({
          connection_config: {
            deployment_mode: "bundled",
            bundle_manifest_path: "/deployments/secrets-manager/bundle.json",
          },
        })}
      />,
    );

    expect(screen.getByLabelText("bundle_manifest_path")).toHaveValue(
      "/deployments/secrets-manager/bundle.json",
    );
    expect(
      screen.getByText(
        /Stages the manifest \+ bundled binaries into the desktop app/i,
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: "Generate Desktop Wrapper" }),
    );
    await waitFor(() => {
      expect(runPipelineWithConfigMock).toHaveBeenCalledWith(
        expect.objectContaining({
          scenarioName: "secrets-manager",
          templateType: TemplateType.BASIC,
          deploymentMode: DeploymentMode.BUNDLED,
          bundleManifestPath: "/deployments/secrets-manager/bundle.json",
          stopAfterStage: StageName.GENERATE,
        }),
      );
    });
  });

  it("uses the saved Vrooli proxy URL for a thin-client wrapper", async () => {
    render(
      <GenerateDesktopButton
        scenario={scenario({
          connection_config: {
            deployment_mode: "external-server",
            proxy_url: "https://tier1.example/apps/secrets-manager/proxy/",
          },
        })}
      />,
    );

    expect(screen.getByLabelText("Proxy URL")).toHaveValue(
      "https://tier1.example/apps/secrets-manager/proxy/",
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Generate Desktop Wrapper" }),
    );

    await waitFor(() => {
      expect(runPipelineWithConfigMock).toHaveBeenCalledWith(
        expect.objectContaining({
          scenarioName: "secrets-manager",
          deploymentMode: DeploymentMode.PROXY,
          proxyUrl: "https://tier1.example/apps/secrets-manager/proxy/",
          stopAfterStage: StageName.GENERATE,
        }),
      );
    });
  });

  it("probes a remote desktop target and makes local-runtime consent explicit", async () => {
    probeEndpointsMock.mockResolvedValue({
      server: { status: "ok" },
      api: { status: "error", message: "API tunnel denied" },
    });
    render(
      <GenerateDesktopButton
        scenario={scenario({
          connection_config: {
            deployment_mode: "external-server",
            proxy_url: "https://tier1.example/apps/secrets-manager/proxy/",
          },
        })}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Test connection" }));
    await waitFor(() => {
      expect(probeEndpointsMock).toHaveBeenCalledWith({
        proxy_url: "https://tier1.example/apps/secrets-manager/proxy/",
      });
    });
    expect(
      await screen.findByText("Connectivity snapshot"),
    ).toBeInTheDocument();
    expect(screen.getByText(/API URL: API tunnel denied/)).toBeInTheDocument();
    const localRuntime = screen.getByLabelText(
      /Let the desktop build run the scenario locally/,
    );
    expect(screen.getByPlaceholderText("vrooli")).toBeDisabled();
    fireEvent.click(localRuntime);
    expect(screen.getByPlaceholderText("vrooli")).not.toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("vrooli"), {
      target: { value: "/usr/local/bin/vrooli" },
    });
  });

  it("shows saved target details, supports editing, reset, and regeneration", async () => {
    render(
      <GenerateDesktopButton
        scenario={scenario({
          has_desktop: true,
          desktop_path: "/artifacts/secrets-manager.AppImage",
          connection_config: {
            deployment_mode: "external-server",
            proxy_url: "https://tier1.example/apps/secrets-manager/proxy/",
          },
        })}
      />,
    );
    expect(screen.getByText("Currently targeting")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Regenerate wrapper" }));
    await waitFor(() => {
      expect(runPipelineWithConfigMock).toHaveBeenCalled();
    });
    fireEvent.click(screen.getByRole("button", { name: "Edit connection" }));
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    expect(screen.getByLabelText("Proxy URL")).toHaveValue("");
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.getByText("Currently targeting")).toBeInTheDocument();
  });
});
