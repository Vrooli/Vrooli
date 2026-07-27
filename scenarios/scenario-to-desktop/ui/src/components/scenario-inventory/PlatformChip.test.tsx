import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  PlatformBuildResultSchema,
  PlatformBuildStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { PlatformChip } from "./PlatformChip";

const mocks = vi.hoisted(() => ({
  getDownloadUrl: vi.fn(),
  triggerDownload: vi.fn(),
  writeToClipboard: vi.fn(),
}));

vi.mock("../../lib/api", () => ({ getDownloadUrl: mocks.getDownloadUrl }));
vi.mock("../../lib/browser", () => ({
  triggerDownload: mocks.triggerDownload,
  writeToClipboard: mocks.writeToClipboard,
}));

function result(status: PlatformBuildStatus, overrides = {}) {
  return create(PlatformBuildResultSchema, {
    platform: Platform.LINUX,
    status,
    errorLog: [],
    ...overrides,
  });
}

describe("PlatformChip", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    mocks.getDownloadUrl.mockReturnValue("/downloads/demo/linux");
    mocks.writeToClipboard.mockResolvedValue({ success: true });
  });

  it("renders pending, building, and skipped states as disabled status controls", () => {
    const { rerender } = render(
      <PlatformChip platform={Platform.LINUX} scenarioName="demo" />,
    );
    expect(
      screen.getByRole("button", { name: "Linux build pending" }),
    ).toBeDisabled();

    rerender(
      <PlatformChip
        platform={Platform.LINUX}
        scenarioName="demo"
        result={result(PlatformBuildStatus.BUILDING)}
      />,
    );
    expect(
      screen.getByRole("button", { name: "Linux build building" }),
    ).toBeDisabled();

    rerender(
      <PlatformChip
        platform={Platform.LINUX}
        scenarioName="demo"
        result={result(PlatformBuildStatus.SKIPPED, {
          skipReason: "Not supported",
        })}
      />,
    );
    expect(screen.getByText("Not supported")).toBeInTheDocument();
  });

  it("downloads a ready artifact through the browser seam", () => {
    render(
      <PlatformChip
        platform={Platform.LINUX}
        scenarioName="demo"
        result={result(PlatformBuildStatus.READY, { fileSize: 2048n })}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Download Linux build" }),
    );
    expect(mocks.getDownloadUrl).toHaveBeenCalledWith("demo", "linux");
    expect(mocks.triggerDownload).toHaveBeenCalledWith({
      url: "/downloads/demo/linux",
    });
    expect(screen.getByText("2 KB")).toBeInTheDocument();
  });

  it("auto-expands a failure, persists visibility, and copies diagnostic logs", async () => {
    render(
      <PlatformChip
        platform={Platform.LINUX}
        scenarioName="demo"
        result={result(PlatformBuildStatus.FAILED, {
          errorLog: ["first failure", "second failure"],
        })}
      />,
    );

    expect(screen.getByText("first failure")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));
    await waitFor(() => {
      expect(mocks.writeToClipboard).toHaveBeenCalledWith(
        "first failure\n\n---\n\nsecond failure",
      );
    });
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Hide Error" }));
    expect(screen.queryByText("first failure")).not.toBeInTheDocument();
    expect(sessionStorage.getItem("error-expanded-demo-3")).toBe("false");
  });
});
