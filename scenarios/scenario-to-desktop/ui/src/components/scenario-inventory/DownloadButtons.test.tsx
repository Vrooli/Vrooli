import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DownloadButtons } from "./DownloadButtons";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { triggerDownload, writeToClipboard } from "../../lib/browser";

const mocks = vi.hoisted(() => ({ getDownloadUrl: vi.fn() }));
vi.mock("../../lib/api", () => ({ getDownloadUrl: mocks.getDownloadUrl }));
vi.mock("../../lib/browser", async (importOriginal) => ({ ...(await importOriginal<typeof import("../../lib/browser")>()), triggerDownload: vi.fn(), writeToClipboard: vi.fn() }));

const artifacts = [
  { platform: "linux", file_name: "canvas.AppImage", size_bytes: 2048, relative_path: "dist/canvas.AppImage" },
  { platform: "win", file_name: "canvas.exe", size_bytes: 1024 },
] as const;

describe("DownloadButtons", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getDownloadUrl.mockReturnValue("/download/canvas/linux");
    vi.mocked(writeToClipboard).mockResolvedValue({ success: true });
  });

  it("does not render when no supported desktop artifacts exist", () => {
    renderWithProviders(<DownloadButtons scenarioName="canvas" artifacts={[]} />);
    expect(screen.queryByText("Included files")).not.toBeInTheDocument();
  });

  it("switches platform tabs and downloads the selected installer", () => {
    renderWithProviders(<DownloadButtons scenarioName="canvas" artifacts={[...artifacts]} />);
    expect(screen.getByRole("button", { name: "🪟Windows" })).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: "🐧Linux" }));
    expect(screen.getByRole("button", { name: "Download Linux installer" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "🪟Windows" }));
    expect(screen.getByRole("button", { name: "Download Windows installer" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Download Windows installer" }));
    expect(mocks.getDownloadUrl).toHaveBeenCalledWith("canvas", "win");
    expect(triggerDownload).toHaveBeenCalledWith({ url: "/download/canvas/linux" });
  });

  it("copies an artifact relative path with a named control", async () => {
    renderWithProviders(<DownloadButtons scenarioName="canvas" artifacts={[...artifacts]} />);
    fireEvent.click(screen.getByRole("button", { name: "🐧Linux" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy canvas.AppImage path" }));
    await waitFor(() => {
      expect(writeToClipboard).toHaveBeenCalledWith("dist/canvas.AppImage");
    });
  });
});
