import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { AutoFixPanel } from "./AutoFixPanel";
import { InvestigationProgress } from "./InvestigationProgress";
import { InvestigationReport } from "./InvestigationReport";
import { DeploymentProgressView, getStepStatusFromProgress } from "./DeploymentProgress";
import { ValidationSummary } from "./ValidationSummary";
import type { Investigation } from "../../types/investigation";
import type { DeploymentManifest } from "../../types/deployment";
import type { ValidationIssue } from "../../lib/api";
import type { DeploymentProgress } from "../../types/progress";

// provider-free-exception: these wizard status views accept all state via props.
describe("wizard status views", () => {
  let clipboardWriteText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: clipboardWriteText } });
  });

  const issues: ValidationIssue[] = [
    { path: "target.host", message: "Host is required", hint: "Use a DNS name", severity: "error" },
    { path: "target.port", message: "Port is unusual", severity: "warn" },
  ];
  const manifest = { target: { host: "old.example", port: 80 } } as unknown as DeploymentManifest;

  it("renders validation states, expands issues, and opens JSON view", () => {
    const onViewInJson = vi.fn();
    const { rerender } = render(<ValidationSummary issues={null} error={null} isValidating={false} />);
    expect(screen.queryByText("Manifest is valid")).not.toBeInTheDocument();
    rerender(<ValidationSummary issues={null} error={null} isValidating />);
    expect(screen.getByText("Validating...")).toBeInTheDocument();
    rerender(<ValidationSummary issues={issues} error={null} isValidating={false} onViewInJson={onViewInJson} />);
    expect(screen.getByText("1 error, 1 warning")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Show details/ }));
    expect(screen.getByText("Host is required")).toBeInTheDocument();
    expect(screen.getByText("Use a DNS name")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Hide details/ }));
    fireEvent.click(screen.getByRole("button", { name: /View in JSON/ }));
    expect(onViewInJson).toHaveBeenCalledOnce();
    rerender(<ValidationSummary issues={[]} error={null} isValidating={false} />);
    expect(screen.getByText("Manifest is valid")).toBeInTheDocument();
    rerender(<ValidationSummary issues={null} error="Validation service unavailable" isValidating={false} />);
    expect(screen.getByText("Validation service unavailable")).toBeInTheDocument();
  });

  it("shows only changed fixable values and applies all fixes", () => {
    const onApplyAll = vi.fn();
    const normalized = { target: { host: "new.example", port: 80 } } as unknown as DeploymentManifest;
    const { rerender } = render(
      <AutoFixPanel issues={issues} normalizedManifest={normalized} currentManifest={manifest} onApplyAll={onApplyAll} />,
    );
    expect(screen.getByText("1 issue can be auto-fixed")).toBeInTheDocument();
    expect(screen.getByText('"old.example"')).toBeInTheDocument();
    expect(screen.getByText('"new.example"')).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Apply All Fixes/ }));
    expect(onApplyAll).toHaveBeenCalledOnce();
    rerender(<AutoFixPanel issues={null} normalizedManifest={normalized} currentManifest={manifest} onApplyAll={onApplyAll} />);
    expect(screen.queryByText(/auto-fixed/)).not.toBeInTheDocument();
  });

  const investigation = (status: Investigation["status"]): Investigation => ({
    id: "investigation-123456",
    deployment_id: "deployment-1",
    status,
    progress: 45,
    findings: status === "completed" ? "Root cause found" : undefined,
    error_message: status === "failed" ? "Agent failed" : undefined,
    details: { source: "agent", operation_mode: "investigate", trigger_reason: "health check" },
    created_at: "2026-08-14T12:00:00.000Z",
    updated_at: "2026-08-14T12:01:00.000Z",
  });

  it("renders investigation lifecycle states and actions", () => {
    const onViewReport = vi.fn();
    const onStop = vi.fn();
    const { rerender } = render(<InvestigationProgress investigation={null} onViewReport={onViewReport} />);
    expect(screen.queryByText("Deployment Investigation")).not.toBeInTheDocument();
    rerender(<InvestigationProgress investigation={investigation("running")} isRunning onStop={onStop} />);
    expect(screen.getByText("Agent investigating VPS...")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    expect(onStop).toHaveBeenCalledOnce();
    rerender(<InvestigationProgress investigation={investigation("completed")} onViewReport={onViewReport} isOutdated />);
    fireEvent.click(screen.getByRole("button", { name: "View Report" }));
    expect(onViewReport).toHaveBeenCalledWith("investigation-123456");
    rerender(<InvestigationProgress investigation={{ ...investigation("failed"), findings: undefined, error_message: undefined }} />);
    expect(screen.getByText(/Investigation failed without details/)).toBeInTheDocument();
    rerender(<InvestigationProgress investigation={{ ...investigation("cancelled"), progress: 20 }} />);
    expect(screen.getByText("Investigation cancelled")).toBeInTheDocument();
  });

  it("renders deployment progress errors, connectivity, and completion", () => {
    const progress: DeploymentProgress = {
      currentStep: "upload",
      currentStepTitle: "Uploading bundle",
      progress: 42,
      steps: [
        { id: "bundle_build", title: "Building bundle", status: "completed" },
        { id: "upload", title: "Uploading bundle", status: "running" },
        { id: "extract", title: "Extracting bundle", status: "failed" },
      ],
      error: "Upload failed",
      isComplete: false,
    };
    render(<DeploymentProgressView progress={progress} isConnected={false} connectionError="Connection lost" />);
    expect(screen.getByText("Deployment Failed")).toBeInTheDocument();
    expect(screen.getByText("Connection lost")).toBeInTheDocument();
    expect(screen.getByText("Reconnecting...")).toBeInTheDocument();
    expect(getStepStatusFromProgress(progress, "upload")).toBe("running");
    expect(getStepStatusFromProgress(null, "upload")).toBe("pending");
    const complete: DeploymentProgress = { ...progress, error: undefined, progress: 100, isComplete: true };
    render(<DeploymentProgressView progress={complete} isConnected connectionError={null} />);
  });

  it("copies findings and submits selected fix permissions from a report", async () => {
    const onClose = vi.fn();
    const onApplyFixes = vi.fn().mockResolvedValue(undefined);
    const report: Investigation = {
      ...investigation("completed"),
      agent_run_id: "agent-run-1",
      details: {
        source: "agent",
        operation_mode: "investigate",
        trigger_reason: "failed health check",
        deployment_step: "verify",
        duration_seconds: 61,
        tokens_used: 1234,
        cost_estimate: 0.42,
      },
    };
    render(<InvestigationReport investigation={report} onClose={onClose} onApplyFixes={onApplyFixes} isOutdated />);
    expect(screen.getByText("Investigation Report")).toBeInTheDocument();
    expect(screen.getByText("Root cause found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Copy Report/ }));
    await waitFor(() => expect(clipboardWriteText).toHaveBeenCalledWith("Root cause found"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "check caddy" } });
    fireEvent.click(screen.getByRole("button", { name: /Apply Selected Fixes/ }));
    await waitFor(() => expect(onApplyFixes).toHaveBeenCalledWith("investigation-123456", {
      immediate: true,
      permanent: false,
      prevention: false,
      note: "check caddy",
    }));
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("renders failed, cancelled, empty, and fix-application reports", () => {
    const base = { ...investigation("failed"), findings: undefined, error_message: undefined };
    const { rerender } = render(<InvestigationReport investigation={base} onClose={vi.fn()} />);
    expect(screen.getByText("Investigation Failed")).toBeInTheDocument();
    expect(screen.getByText("No findings available.")).toBeInTheDocument();
    rerender(<InvestigationReport investigation={{ ...base, status: "cancelled", error_message: undefined }} onClose={vi.fn()} />);
    expect(screen.getByText("Investigation Cancelled")).toBeInTheDocument();
    expect(screen.getByText("No findings available.")).toBeInTheDocument();
    rerender(<InvestigationReport investigation={{
      ...base,
      status: "completed",
      findings: "Fix applied",
      details: { operation_mode: "fix-application", source_findings: "Original finding", source: "investigation", trigger_reason: "manual", duration_seconds: 0, tokens_used: 0, cost_estimate: 0 },
    }} onClose={vi.fn()} />);
    expect(screen.getByText("Fix Application Report")).toBeInTheDocument();
    expect(screen.getByText("Original finding")).toBeInTheDocument();
    expect(screen.getByText("Fix Application Results")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Apply Selected Fixes/ })).not.toBeInTheDocument();
  });
});
