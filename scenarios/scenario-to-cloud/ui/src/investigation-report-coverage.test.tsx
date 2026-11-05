import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";
import { InvestigationReport } from "./components/wizard/InvestigationReport";

const investigation = {
  id: "inv-1", status: "completed", created_at: "2026-08-14T00:00:00Z", findings: "all good",
  error_message: "warning", agent_run_id: "agent-1",
  details: { duration_seconds: 125, tokens_used: 1234, cost_estimate: 0.12, operation_mode: "auto-fix", trigger_reason: "failed deploy", deployment_step: "setup" },
};

describe("investigation report states", () => {
  beforeEach(() => Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } }));

  it("shows findings and applies selected fixes", async () => {
    const onApply = vi.fn().mockResolvedValue(undefined); const close = vi.fn();
    renderWithProviders(<InvestigationReport investigation={investigation as any} onClose={close} onApplyFixes={onApply} isOutdated />);
    fireEvent.click(screen.getByRole("button", { name: "Copy Report" }));
    fireEvent.click(screen.getByText("Permanent Fix"));
    fireEvent.click(screen.getByText("Prevention", { exact: true }));
    fireEvent.change(screen.getByPlaceholderText(/additional context/), { target: { value: "carefully" } });
    fireEvent.click(screen.getByRole("button", { name: "Apply Selected Fixes" }));
    await waitFor(() => expect(onApply).toHaveBeenCalledWith("inv-1", expect.objectContaining({ permanent: true, prevention: true, note: "carefully" })));
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(close).toHaveBeenCalled();
  });

  it("renders failed, cancelled, fix-report, and empty-finding states", () => {
    const { rerender } = renderWithProviders(<InvestigationReport investigation={{ ...investigation, status: "failed", findings: undefined, error_message: "nope" } as any} onClose={vi.fn()} />);
    expect(screen.getByText("Failed")).toBeInTheDocument();
    rerender(<InvestigationReport investigation={{ ...investigation, status: "cancelled", findings: undefined, error_message: undefined } as any} onClose={vi.fn()} />);
    expect(screen.getByText("Cancelled")).toBeInTheDocument();
    rerender(<InvestigationReport investigation={{ ...investigation, details: { ...investigation.details, operation_mode: "fix-application", source_findings: "original" } } as any} onClose={vi.fn()} />);
    expect(screen.getByText("Original Investigation Findings")).toBeInTheDocument();
  });
});
