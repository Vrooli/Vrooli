import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { ScenarioDetails } from "./ScenarioDetails";
import type { ScenarioDesktopStatus } from "./types";

vi.mock("./GenerateDesktopButton", () => ({
  GenerateDesktopButton: ({
    scenario,
  }: {
    scenario: ScenarioDesktopStatus;
  }) => <div>Generate desktop for {scenario.name}</div>,
}));

vi.mock("./BuildDesktopButton", () => ({
  BuildDesktopButton: ({ scenarioName }: { scenarioName: string }) => (
    <div>Build installers for {scenarioName}</div>
  ),
}));

vi.mock("./DownloadButtons", () => ({
  DownloadButtons: ({ scenarioName }: { scenarioName: string }) => (
    <div>Download builds for {scenarioName}</div>
  ),
}));

vi.mock("./TelemetryUploadCard", () => ({
  TelemetryUploadCard: ({ appDisplayName }: { appDisplayName?: string }) => (
    <div>Telemetry for {appDisplayName}</div>
  ),
}));

vi.mock("./DeleteButton", () => ({
  DeleteButton: ({ scenarioName }: { scenarioName: string }) => (
    <div>Delete desktop for {scenarioName}</div>
  ),
}));

function scenario(
  overrides: Partial<ScenarioDesktopStatus> = {},
): ScenarioDesktopStatus {
  return {
    name: "secrets-manager",
    display_name: "Secrets Manager",
    has_desktop: false,
    ...overrides,
  };
}

function stepContainer(title: string) {
  const container = screen.getByText(title).closest("div.rounded-xl");
  if (!container) {
    throw new Error(`Could not find the step card for ${title}`);
  }
  return container;
}

describe("ScenarioDetails", () => {
  it("guides an operator to generate a missing desktop wrapper before build and download", () => {
    const onClose = vi.fn();

    render(<ScenarioDetails scenario={scenario()} onClose={onClose} />);

    expect(screen.getByText("Needs desktop wrapper")).toBeInTheDocument();
    expect(
      screen.getByText("Generate desktop for secrets-manager"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Finish step 1 so we know which scenario to build from.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Build at least one installer to unlock downloads and telemetry uploads.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Wrapper path: Not generated yet"),
    ).toBeInTheDocument();
    expect(stepContainer("Build installers")).toHaveClass("opacity-50");
    expect(stepContainer("Download your build")).toHaveClass("opacity-50");

    fireEvent.click(screen.getByRole("button", { name: "Back to list" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("unlocks builds and downloads after the wrapper and artifacts are available", () => {
    render(
      <ScenarioDetails
        scenario={scenario({
          has_desktop: true,
          desktop_path: "/tmp/secrets-manager-desktop",
          version: "1.2.3",
          build_artifacts: [
            {
              platform: "linux",
              file_name: "secrets-manager.AppImage",
              size_bytes: 1024,
            },
          ],
        })}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("Desktop wrapper ready")).toBeInTheDocument();
    expect(
      screen.getByText("Build installers for secrets-manager"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Download builds for secrets-manager"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Telemetry for Secrets Manager"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Wrapper path: /tmp/secrets-manager-desktop"),
    ).toBeInTheDocument();
    expect(screen.getByText("Current version: 1.2.3")).toBeInTheDocument();
    expect(stepContainer("Build installers")).toHaveClass("border-green-700");
    expect(stepContainer("Download your build")).toHaveClass(
      "border-green-700",
    );
  });
});
