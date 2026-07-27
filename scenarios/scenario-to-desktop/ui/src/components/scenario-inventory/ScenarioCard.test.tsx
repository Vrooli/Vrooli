import { fireEvent, render, screen } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ScenarioCard } from "./ScenarioCard";
import type { ScenarioDesktopStatus } from "./types";

const mocks = vi.hoisted(() => ({
  getDownloadUrl: vi.fn(),
  triggerDownload: vi.fn(),
}));

vi.mock("../../lib/api", () => ({ getDownloadUrl: mocks.getDownloadUrl }));
vi.mock("../../lib/browser", () => ({ triggerDownload: mocks.triggerDownload }));

function scenario(overrides: Partial<ScenarioDesktopStatus> = {}): ScenarioDesktopStatus {
  return {
    name: "canvas-lab",
    has_desktop: true,
    built: true,
    desktop_path: "/desktop/canvas-lab",
    build_artifacts: [],
    ...overrides,
  };
}

describe("ScenarioCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getDownloadUrl.mockReturnValue("/downloads/canvas-lab/linux");
  });

  it("opens the desktop builder through an explicit control", () => {
    const onSelect = vi.fn();
    const item = scenario({ display_name: "Canvas Lab", package_size: 2048 });
    render(<ScenarioCard scenario={item} onSelect={onSelect} />);

    expect(screen.getByText("Built")).toBeInTheDocument();
    expect(screen.getByText("2 KB")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open desktop builder" }));
    expect(onSelect).toHaveBeenCalledWith(item);
  });

  it("keeps platform downloads independent from opening the builder", () => {
    const onSelect = vi.fn();
    render(
      <ScenarioCard
        scenario={scenario({
          build_artifacts: [
            { platform: "linux", file_name: "canvas.AppImage" },
            { platform: "linux", file_name: "canvas.deb" },
          ],
        })}
        onSelect={onSelect}
      />,
    );

    const downloadButtons = screen.getAllByRole("button", { name: /🐧/ });
    const downloadButton = downloadButtons[0];
    if (!downloadButton) throw new Error("expected a Linux download control");
    fireEvent.click(downloadButton);

    expect(mocks.getDownloadUrl).toHaveBeenCalledWith("canvas-lab", "linux");
    expect(mocks.triggerDownload).toHaveBeenCalledWith({ url: "/downloads/canvas-lab/linux" });
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("labels a scenario that has not yet generated a desktop wrapper", () => {
    render(
      <ScenarioCard
        scenario={scenario({
          has_desktop: false,
          built: false,
          desktop_path: undefined,
        })}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText("Not Generated")).toBeInTheDocument();
    expect(screen.getByText("Desktop wrapper not generated yet")).toBeInTheDocument();
  });
});
