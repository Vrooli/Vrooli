import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  CheckStatus,
  PreflightCheckStep,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { renderWithProviders } from "../../../test-utils/renderWithProviders";
import { usePipelineStore } from "../../../store";
import { PreflightSection } from "./PreflightSection";

describe("PreflightSection", () => {
  beforeEach(() => {
    usePipelineStore.getState().reset();
  });

  it("explains that preflight is unnecessary for non-bundled desktop deployments", () => {
    renderWithProviders(
      <PreflightSection scenarioName="desktop-app" isBundled={false} />,
    );

    expect(
      screen.getByText(
        "Preflight validation is only required for bundled deployment mode.",
      ),
    ).toBeInTheDocument();
  });

  it("blocks a bundled preflight until the operator provides a bundle manifest", async () => {
    const user = userEvent.setup();
    renderWithProviders(<PreflightSection scenarioName="desktop-app" />);

    expect(
      screen.getByText(
        "Configure the bundle manifest path in the Configuration section above to enable preflight checks.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Run preflight" }),
    ).toBeDisabled();

    await user.click(
      screen.getByRole("button", { name: "Show preflight JSON" }),
    );

    expect(screen.getByText("Preflight JSON")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Copy preflight JSON" }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Download preflight JSON" }),
    ).toBeDisabled();
  });

  it("enables preflight when a bundled manifest is supplied", () => {
    renderWithProviders(
      <PreflightSection
        scenarioName="desktop-app"
        bundleManifestPath="/tmp/desktop-app/bundle-manifest.json"
      />,
    );

    expect(screen.getByRole("button", { name: "Run preflight" })).toBeEnabled();
    expect(
      screen.getByText("/tmp/desktop-app/bundle-manifest.json"),
    ).toBeInTheDocument();
  });

  it("shows resource eligibility and a non-promotable warning from the resolved target plan", () => {
    usePipelineStore.setState({
      pipelineStatus: {
        stages: {
          resolveDeployment: {
            status: StageStatus.COMPLETED,
            details: {
              kind: {
                case: "resolveDeployment",
                value: {
                  promotable: false,
                  resources: [
                    {
                      requestedResource: "vault",
                      os: "windows",
                      architecture: "amd64",
                      bundling: "required",
                      privilege: "none",
                      eligibility: "ineligible",
                      eligibilityReason:
                        "Windows credential store is unavailable",
                      requires: ["secret-tool"],
                    },
                  ],
                  hostRequirements: [
                    {
                      name: "secret-tool",
                      kind: "tool",
                      os: "windows",
                      bundling: "host-required",
                      privilege: "user",
                      required: true,
                      verdict: "ineligible",
                      reason: "required host tool is absent on windows",
                    },
                  ],
                },
              },
            } as never,
          },
        },
      } as never,
    });

    renderWithProviders(<PreflightSection scenarioName="desktop-app" />);

    expect(
      screen.getByText("Target deployment eligibility"),
    ).toBeInTheDocument();
    expect(screen.getByText("Non-promotable")).toBeInTheDocument();
    expect(
      screen.getByText("Windows credential store is unavailable"),
    ).toBeInTheDocument();
    expect(screen.getByText("Requires: secret-tool")).toBeInTheDocument();
    expect(screen.getByText("Host requirements")).toBeInTheDocument();
    expect(screen.getByText("tool: secret-tool")).toBeInTheDocument();
    expect(
      screen.getByText("required host tool is absent on windows"),
    ).toBeInTheDocument();
  });

  it("shows a running preflight honestly and lets the operator cancel it", async () => {
    const user = userEvent.setup();
    const cancelPipeline = vi.fn().mockResolvedValue(undefined);
    usePipelineStore.setState({
      runStatus: "running",
      cancelPipeline,
      pipelineStatus: {
        stages: { preflight: { status: StageStatus.RUNNING } },
      } as never,
    });

    renderWithProviders(
      <PreflightSection
        scenarioName="desktop-app"
        bundleManifestPath="/tmp/desktop-app/bundle-manifest.json"
      />,
    );

    expect(screen.getByRole("button", { name: "Running..." })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(cancelPipeline).toHaveBeenCalledOnce();
    expect(
      screen.getByText(/Starting the runtime supervisor/),
    ).toBeInTheDocument();
  });

  it("renders actionable failed-preflight evidence and allows an explicit override", async () => {
    const user = userEvent.setup();
    const setPreflightOverride = vi.fn();
    usePipelineStore.setState({
      runStatus: "failed",
      errorInfo: { message: "runtime did not become ready" } as never,
      preflightOverride: false,
      setPreflightOverride,
      pipelineStatus: {
        stages: {
          bundle: { status: StageStatus.FAILED, error: "missing staged asset" },
        },
      } as never,
    });

    renderWithProviders(
      <PreflightSection
        scenarioName="desktop-app"
        bundleManifestPath="/tmp/desktop-app/bundle-manifest.json"
      />,
    );

    expect(
      screen.getAllByText("runtime did not become ready").length,
    ).toBeGreaterThan(0);
    expect(
      screen.getByText(
        "Validation did not complete. Review the error above and re-run preflight.",
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByLabelText("Override"));
    expect(setPreflightOverride).toHaveBeenCalledWith(true);
  });

  it("renders successful runtime, readiness, diagnostics, and check evidence", () => {
    usePipelineStore.setState({
      runStatus: "completed",
      preflightResult: {
        validation: {
          valid: true,
          errors: [],
          warnings: [],
          missingBinaries: [],
          missingAssets: [],
          invalidChecksums: [],
        },
        ready: {
          ready: false,
          details: [
            {
              serviceId: "api",
              ready: false,
              message: "waiting",
              updatedAt: { seconds: 2n, nanos: 0 },
            },
          ],
          waitedSeconds: 2,
          snapshotAt: { seconds: 2n, nanos: 0 },
        },
        ports: [{ serviceId: "api", name: "http", port: 15200 }],
        telemetry: { path: "/tmp/telemetry.json" },
        runtime: { instanceId: "runtime-1", dryRun: false },
        serviceFingerprints: [
          {
            serviceId: "api",
            platform: "linux",
            binaryPath: "/usr/bin/api",
            binarySizeBytes: 42n,
          },
        ],
        logTails: [{ serviceId: "api", lines: 1, content: "starting" }],
        checks: [
          {
            id: "validation",
            step: PreflightCheckStep.VALIDATION,
            name: "Manifest",
            status: CheckStatus.PASSED,
          },
          {
            id: "runtime",
            step: PreflightCheckStep.RUNTIME,
            name: "Runtime",
            status: CheckStatus.PASSED,
          },
          {
            id: "services",
            step: PreflightCheckStep.SERVICES,
            name: "API",
            status: CheckStatus.FAILED,
          },
          {
            id: "diagnostics",
            step: PreflightCheckStep.DIAGNOSTICS,
            name: "Logs",
            status: CheckStatus.PASSED,
          },
        ],
        errors: [{ message: "optional service unavailable" }],
      } as never,
      pipelineStatus: {
        stages: {
          bundle: { status: StageStatus.COMPLETED },
          preflight: { status: StageStatus.COMPLETED },
        },
      } as never,
    });

    renderWithProviders(
      <PreflightSection
        scenarioName="desktop-app"
        bundleManifestPath="/tmp/desktop-app/bundle-manifest.json"
        bundleManifest={{ services: [] }}
      />,
    );

    expect(
      screen.getByText("No validation issues detected."),
    ).toBeInTheDocument();
    expect(screen.getByText("Preflight warnings")).toBeInTheDocument();
    expect(screen.getByText("Readiness details")).toBeInTheDocument();
    expect(
      screen.getByText(/Waited 2s before capturing status/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Control API is responding. Runtime supervisor initialized.",
      ),
    ).toBeInTheDocument();
  });
});
