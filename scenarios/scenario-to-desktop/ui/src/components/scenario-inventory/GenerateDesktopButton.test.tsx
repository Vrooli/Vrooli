import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GenerateDesktopButton } from "./GenerateDesktopButton";
import type { ScenarioDesktopStatus } from "./types";

const { resetMock, runPipelineWithConfigMock } = vi.hoisted(() => ({
  resetMock: vi.fn(),
  runPipelineWithConfigMock: vi.fn(),
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

function scenario(overrides: Partial<ScenarioDesktopStatus> = {}): ScenarioDesktopStatus {
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
      />
    );

    expect(screen.getByLabelText("bundle_manifest_path")).toHaveValue("/deployments/secrets-manager/bundle.json");
    expect(screen.getByText(/Stages the manifest \+ bundled binaries into the desktop app/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Generate Desktop Wrapper" }));
    await waitFor(() => {
      expect(runPipelineWithConfigMock).toHaveBeenCalledWith(
        expect.objectContaining({
          scenario_name: "secrets-manager",
          template_type: "universal",
          deployment_mode: "bundled",
          bundle_manifest_path: "/deployments/secrets-manager/bundle.json",
          stop_after_stage: "generate",
        })
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
      />
    );

    expect(screen.getByLabelText("Proxy URL")).toHaveValue("https://tier1.example/apps/secrets-manager/proxy/");
    fireEvent.click(screen.getByRole("button", { name: "Generate Desktop Wrapper" }));

    await waitFor(() => {
      expect(runPipelineWithConfigMock).toHaveBeenCalledWith(
        expect.objectContaining({
          scenario_name: "secrets-manager",
          deployment_mode: "external-server",
          proxy_url: "https://tier1.example/apps/secrets-manager/proxy/",
          stop_after_stage: "generate",
        })
      );
    });
  });
});
