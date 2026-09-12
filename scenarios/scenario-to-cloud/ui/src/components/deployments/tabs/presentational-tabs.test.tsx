import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { FileTree } from "./FileTree";
import { FileViewer } from "./FileViewer";
import { LogViewer } from "./LogViewer";
import { PortTable } from "./PortTable";
import { QuickAccess } from "./QuickAccess";
import { Timeline } from "./Timeline";
import { ConfirmationDialog } from "./ConfirmationDialog";
import type { FileEntry, HistoryEvent, LogEntry, PortBinding } from "../../../lib/api";

// provider-free-exception: these presentational deployment tabs have no context dependencies.
describe("presentational deployment tabs", () => {
  let clipboardWriteText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText: clipboardWriteText },
    });
  });

  it("renders quick access files and reports the selected path", () => {
    const onSelect = vi.fn();
    render(<QuickAccess deploymentId="deployment-1" onSelect={onSelect} selectedPath=".env" />);

    expect(screen.getByRole("heading", { name: "Quick Access" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "manifest.json" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: ".env" })).toHaveClass("text-blue-400");
    fireEvent.click(screen.getByRole("button", { name: "Caddyfile" }));
    expect(onSelect).toHaveBeenCalledWith("/etc/caddy/Caddyfile");
  });

  it("requires the confirmation phrase before invoking a destructive action", () => {
    const onInputChange = vi.fn();
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const { rerender } = render(
      <ConfirmationDialog
        title="Destroy instance"
        description="This cannot be undone."
        confirmText="DESTROY"
        inputValue=""
        onInputChange={onInputChange}
        onConfirm={onConfirm}
        onCancel={onCancel}
        isPending={false}
        isDestructive
      />,
    );
    expect(screen.getByText("This cannot be undone.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("DESTROY"), { target: { value: "DESTROY" } });
    expect(onInputChange).toHaveBeenCalledWith("DESTROY");
    rerender(
      <ConfirmationDialog
        title="Destroy instance"
        description="This cannot be undone."
        confirmText="DESTROY"
        inputValue="DESTROY"
        onInputChange={onInputChange}
        onConfirm={onConfirm}
        onCancel={onCancel}
        isPending={false}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onConfirm).toHaveBeenCalledOnce();
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("sorts ports and renders type counts, process metadata, and manifest status", () => {
    const ports: PortBinding[] = [
      { port: 8080, process: "app", type: "scenario", pid: 42, matches_manifest: true },
      { port: 80, process: "caddy", type: "edge", matches_manifest: false },
      { port: 5432, process: "", type: "resource" },
      { port: 22, process: "sshd", type: "system" },
      { port: 9000, process: "unknown", type: "unexpected" },
    ];
    render(<PortTable ports={ports} />);

    const rows = screen.getAllByRole("row");
    expect(rows[1]).toHaveTextContent("22");
    expect(rows[5]).toHaveTextContent("9000");
    expect(screen.getByText("System: 1")).toBeInTheDocument();
    expect(screen.getByText("Scenario: 1")).toBeInTheDocument();
    expect(screen.getByText("(PID: 42)")).toBeInTheDocument();
    expect(screen.getByText("Expected")).toBeInTheDocument();
    expect(screen.getAllByText("Unexpected")).toHaveLength(2);

    render(<PortTable ports={[]} />);
    expect(screen.getByText("No listening ports detected")).toBeInTheDocument();
  });

  it("navigates files and chooses the correct icon branches", () => {
    const onNavigate = vi.fn();
    const onSelectFile = vi.fn();
    const entries: FileEntry[] = [
      { name: "src", type: "directory", size_bytes: 0, modified: "", permissions: "" },
      { name: "README.md", type: "file", size_bytes: 1024, modified: "", permissions: "" },
      { name: "config.json", type: "file", size_bytes: 4, modified: "", permissions: "" },
      { name: "main.go", type: "file", size_bytes: 8, modified: "", permissions: "" },
      { name: "bundle.tar.gz", type: "file", size_bytes: 8, modified: "", permissions: "" },
      { name: "current", type: "symlink", size_bytes: 0, modified: "", permissions: "" },
    ];
    render(
      <FileTree
        entries={entries}
        currentPath="/var/app/logs"
        onNavigate={onNavigate}
        onSelectFile={onSelectFile}
        selectedFile="/var/app/logs/README.md"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: ".." }));
    expect(onNavigate).toHaveBeenCalledWith("/var/app");
    fireEvent.click(screen.getByRole("button", { name: /src/ }));
    expect(onNavigate).toHaveBeenCalledWith("/var/app/logs/src");
    fireEvent.click(screen.getByRole("button", { name: /README.md/ }));
    expect(onSelectFile).toHaveBeenCalledWith("/var/app/logs/README.md");

    render(
      <FileTree entries={[]} currentPath="/" onNavigate={onNavigate} onSelectFile={onSelectFile} selectedFile={null} />,
    );
    expect(screen.getByText("Empty directory")).toBeInTheDocument();
  });

  it("renders file viewer empty, loading, error, binary, and source states", () => {
    const { rerender } = render(
      <FileViewer path={null} content={undefined} isLoading={false} />,
    );
    expect(screen.getByText("Select a file to view its contents")).toBeInTheDocument();

    rerender(<FileViewer path="app.json" content={undefined} isLoading />);
    expect(document.querySelector(".animate-spin")).toBeInTheDocument();
    rerender(<FileViewer path="app.json" content={undefined} isLoading={false} error={new Error("path_not_allowed")} />);
    expect(screen.getByText("Access denied")).toBeInTheDocument();
    rerender(<FileViewer path="app.json" content={undefined} isLoading={false} error={new Error("404 not found")} />);
    expect(screen.getByText("File not found")).toBeInTheDocument();
    rerender(<FileViewer path="app.json" content={undefined} isLoading={false} />);
    expect(screen.getByText("Unable to load file content")).toBeInTheDocument();
    rerender(<FileViewer path="empty.txt" content="" isLoading={false} sizeBytes={0} />);
    expect(screen.getByText("Empty file")).toBeInTheDocument();
    rerender(<FileViewer path="archive.bin" content="[Binary file]" isLoading={false} truncated />);
    expect(screen.getByText("Binary files cannot be previewed")).toBeInTheDocument();
    expect(screen.getByText("File truncated (showing first 1MB)")).toBeInTheDocument();
    rerender(<FileViewer path="main.ts" content="const value = 1;" isLoading={false} sizeBytes={16} />);
    expect(screen.getByText("main.ts")).toBeInTheDocument();
  });

  it("renders timeline history and expands event details", () => {
    const events: HistoryEvent[] = [
      {
        type: "deploy_completed",
        timestamp: "2026-08-14T12:00:00.000Z",
        message: "Deployment ready",
        success: true,
        duration_ms: 1500,
        bundle_hash: "1234567890abcdef-extra",
        data: { host: "example.test" },
      },
      {
        type: "deploy_failed",
        timestamp: "2026-08-14T12:05:00.000Z",
        message: "Deployment failed",
        success: false,
        details: "SSH unavailable",
        step_name: "connect",
      },
    ];
    const { rerender } = render(<Timeline events={events} />);
    expect(screen.getByText("Deployed")).toBeInTheDocument();
    expect(screen.getByText("Deploy Failed")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Deployment ready"));
    expect(screen.getByText(/1234567890abcdef/)).toBeInTheDocument();
    expect(screen.getByText(/example.test/)).toBeInTheDocument();
    fireEvent.click(screen.getByText("Deployment ready"));
    expect(screen.queryByText(/1234567890abcdef/)).not.toBeInTheDocument();
    rerender(<Timeline events={[]} />);
    expect(screen.getByText("No history events yet")).toBeInTheDocument();
  });

  it("copies logs, highlights log content, and restores latest scrolling", async () => {
    const logs: LogEntry[] = [
      { timestamp: "2026-08-14T12:00:00.000Z", source: "api", level: "INFO", message: "GET /health in 12ms" },
      { timestamp: "not-a-date", source: "worker", level: "ERROR", message: "POST /jobs failed" },
      { timestamp: "2026-08-14T12:01:00.000Z", source: "worker", level: "WARN", message: "retry" },
      { timestamp: "2026-08-14T12:02:00.000Z", source: "worker", level: "DEBUG", message: "value=42" },
    ];
    render(<LogViewer logs={logs} total={4} sources={["api", "worker"]} />);
    expect(screen.getByText("4 lines")).toBeInTheDocument();
    expect(screen.getByText("GET")).toBeInTheDocument();
    expect(screen.getByText("/health")).toBeInTheDocument();
    expect(screen.getByText("not-a-date")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Copy/ }));
    await waitFor(() => expect(clipboardWriteText).toHaveBeenCalledWith(expect.stringContaining("GET /health")));

    const checkbox = screen.getByRole("checkbox", { name: "Auto-scroll to latest" });
    fireEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();
    expect(screen.getByRole("button", { name: /Latest/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Latest/ }));
    expect(checkbox).toBeChecked();
    render(<LogViewer logs={[]} total={0} sources={[]} />);
    expect(screen.getByText("No logs available")).toBeInTheDocument();
  });
});
