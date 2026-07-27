import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  FingerprintsPanel,
  LogTailsPanel,
  PortSummaryPanel,
} from "./DiagnosticsPanels";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { writeToClipboard } from "../../lib/browser";

vi.mock("../../lib/browser", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../lib/browser")>()),
  writeToClipboard: vi.fn(),
}));

describe("preflight diagnostics panels", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(writeToClipboard).mockResolvedValue({ success: true });
  });

  it("omits empty diagnostics panels", () => {
    renderWithProviders(
      <>
        <LogTailsPanel logTails={[] as never} />
        <FingerprintsPanel fingerprints={[] as never} />
        <PortSummaryPanel />
      </>,
    );
    expect(screen.queryByText("Service log tails")).not.toBeInTheDocument();
    expect(screen.queryByText(/Service fingerprints/)).not.toBeInTheDocument();
  });

  it("renders log-tail content/errors and copies an individual tail", async () => {
    renderWithProviders(
      <LogTailsPanel
        logTails={
          [
            { service_id: "api", content: "one\ntwo", lines: 10 },
            {
              service_id: "worker",
              content: "",
              lines: 3,
              error: "tail unavailable",
            },
          ] as never
        }
      />,
    );
    expect(screen.getByText("api · 2 of 10 lines")).toBeInTheDocument();
    expect(document.querySelector("pre")?.textContent).toBe("one\ntwo");
    expect(screen.getByText("tail unavailable")).toBeInTheDocument();
    const [copyButton] = screen.getAllByRole("button", { name: "Copy" });
    if (!copyButton) throw new Error("expected log-tail copy control");
    fireEvent.click(copyButton);
    await waitFor(() => {
      expect(writeToClipboard).toHaveBeenCalledWith("one\ntwo");
    });
    expect(
      await screen.findByRole("button", { name: "Copied" }),
    ).toBeInTheDocument();
  });

  it("shows fingerprint provenance and port/telemetry summary", () => {
    renderWithProviders(
      <>
        <FingerprintsPanel
          fingerprints={
            [
              {
                service_id: "api",
                binary_path: "bin/api",
                binary_resolved_path: "/bundle/bin/api",
                platform: "linux",
                binary_size_bytes: 2048,
                binary_mtime: "2026-07-27T12:00:00Z",
                binary_sha256: "0123456789abcdef0123456789abcdef",
              },
              { service_id: "worker", error: "missing binary" },
            ] as never
          }
        />
        <PortSummaryPanel
          portSummary="api=19925"
          telemetryPath="/tmp/telemetry.json"
        />
      </>,
    );
    expect(screen.getByText("bin/api")).toHaveAttribute(
      "title",
      "/bundle/bin/api",
    );
    expect(screen.getByText("0123456789abcdef")).toHaveAttribute(
      "title",
      "0123456789abcdef0123456789abcdef",
    );
    expect(screen.getByText("missing binary")).toBeInTheDocument();
    expect(screen.getByText("Ports: api=19925")).toBeInTheDocument();
    expect(
      screen.getByText("Telemetry: /tmp/telemetry.json"),
    ).toBeInTheDocument();
  });
});
