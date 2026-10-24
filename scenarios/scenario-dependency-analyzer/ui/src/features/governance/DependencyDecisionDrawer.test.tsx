import { create } from "@bufbuild/protobuf";
import { DependencyUsageGroupSchema, SecurityVersionRangeSchema, UpsertApprovedDependencyResponseSchema, VulnerabilityRemediationResponseSchema } from "@vrooli/proto-types/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_pb";
import { fireEvent, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { DependencyDecisionDrawer, VulnerabilityRemediationDrawer } from "./DependencyDecisionDrawer";
import { governanceRecordFromDecision } from "./governanceTypes";

const group = create(DependencyUsageGroupSchema, { ecosystem: "npm", packageName: "example", scenarios: ["demo"] });
const decisionRecord = () => governanceRecordFromDecision({ ecosystem: "npm", packageName: "example", state: "approved", versionRange: "*", rationale: "Reviewed dependency.", approvedBy: "operator" });
const decisionProps = (): ComponentProps<typeof DependencyDecisionDrawer> => ({ group, initialState: "approved", open: true, preview: null, applied: null, busy: false, error: null, onClose: vi.fn(), onPreview: vi.fn(), onApply: vi.fn() });
const remediationPreview = () => create(VulnerabilityRemediationResponseSchema, { found: true, vulnerability: { ecosystem: "npm", packageName: "example", vulnerabilityId: "GHSA-original", aliases: ["CVE-alias"], affectedRanges: [{ range: "<2" }], fixedRanges: [{ range: ">=2" }] } });
const remediationProps = (): ComponentProps<typeof VulnerabilityRemediationDrawer> => ({ group, initialVulnerabilityId: "GHSA-original", open: true, preview: remediationPreview(), result: null, busy: false, error: null, onClose: vi.fn(), onPreview: vi.fn(), onApply: vi.fn() });

describe("reviewed dependency decision binding", () => {
  for (const field of ["Version range", "Rationale", "Reviewer", "Replacement", "Deny"] as const) {
    it(`requires a new preview after changing ${field}`, () => {
      const props = decisionProps();
      const onPreview = vi.fn();
      const view = renderWithProviders(<DependencyDecisionDrawer {...props} onPreview={onPreview} />);
      expect(screen.getByRole("button", { name: "Apply decision" })).toBeDisabled();
      fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Reviewed dependency." } });
      fireEvent.click(screen.getByRole("button", { name: "Dry-run preview" }));
      expect(onPreview).toHaveBeenCalledWith(expect.objectContaining({ packageName: "example", rationale: "Reviewed dependency." }));
      view.rerender(<DependencyDecisionDrawer {...props} onPreview={onPreview} preview={create(UpsertApprovedDependencyResponseSchema, { dryRun: true, record: decisionRecord() })} />);
      expect(screen.getByRole("button", { name: "Apply decision" })).toBeEnabled();
      if (field === "Deny") fireEvent.click(screen.getByRole("radio", { name: "Deny" }));
      else fireEvent.change(screen.getByLabelText(field), { target: { value: field === "Version range" ? "^2.0.0" : "Updated review value" } });
      expect(screen.getByRole("button", { name: "Apply decision" })).toBeDisabled();
      fireEvent.click(screen.getByRole("button", { name: "Apply decision" }));
      expect(props.onApply).not.toHaveBeenCalled();
      fireEvent.click(screen.getByRole("button", { name: "Dry-run preview" }));
      const refreshedRecord = onPreview.mock.calls.at(-1)?.[0];
      expect(refreshedRecord).toBeDefined();
      view.rerender(<DependencyDecisionDrawer {...props} onPreview={onPreview} preview={create(UpsertApprovedDependencyResponseSchema, { dryRun: true, record: refreshedRecord })} />);
      fireEvent.click(screen.getByRole("button", { name: "Apply decision" }));
      expect(props.onApply).toHaveBeenCalledWith(refreshedRecord);
    });
  }

  it("refuses a preview for another dependency and retains close/error behavior", () => {
    const props = decisionProps();
    const view = renderWithProviders(<DependencyDecisionDrawer {...props} preview={create(UpsertApprovedDependencyResponseSchema, { dryRun: true, record: decisionRecord() })} />);
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Reviewed dependency." } });
    view.rerender(<DependencyDecisionDrawer {...props} group={create(DependencyUsageGroupSchema, { ecosystem: "npm", packageName: "other" })} preview={create(UpsertApprovedDependencyResponseSchema, { dryRun: true, record: decisionRecord() })} error="Preview was rejected." />);
    expect(screen.getByRole("button", { name: "Apply decision" })).toBeDisabled();
    expect(screen.getByText("Preview was rejected.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close dependency decision" }));
    expect(props.onClose).toHaveBeenCalledOnce();
  });

  it("does not permit a missing rationale or a pending operation", () => {
    const props = decisionProps();
    const view = renderWithProviders(<DependencyDecisionDrawer {...props} />);
    expect(screen.getByRole("button", { name: "Dry-run preview" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Reviewed dependency." } });
    view.rerender(<DependencyDecisionDrawer {...props} busy preview={create(UpsertApprovedDependencyResponseSchema, { dryRun: true, record: decisionRecord() })} />);
    expect(screen.getByRole("button", { name: "Dry-run preview" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Apply decision" })).toBeDisabled();
  });
});

describe("reviewed vulnerability evidence binding", () => {
  it("rejects a changed vulnerability ID but accepts an alias from the same reviewed evidence", () => {
    const props = remediationProps();
    renderWithProviders(<VulnerabilityRemediationDrawer {...props} />);
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Deny reviewed vulnerable range." } });
    expect(screen.getByRole("button", { name: "Apply denied range" })).toBeEnabled();
    fireEvent.change(screen.getByLabelText("Vulnerability ID"), { target: { value: "GHSA-other" } });
    expect(screen.getByRole("button", { name: "Apply denied range" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Apply denied range" }));
    expect(props.onApply).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Preview remediation" }));
    expect(props.onPreview).toHaveBeenCalledWith("GHSA-other");
    fireEvent.change(screen.getByLabelText("Vulnerability ID"), { target: { value: "CVE-alias" } });
    expect(screen.getByRole("button", { name: "Apply denied range" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Apply denied range" }));
    expect(props.onApply).toHaveBeenCalledWith(expect.objectContaining({ vulnerabilityId: "CVE-alias", affectedRange: "<2", fixedRange: ">=2" }));
  });

  it.each(["another-package", "missing-range", "only-fixed-event", "not-found"])("rejects unusable evidence: %s", (kind) => {
    const props = remediationProps();
    const preview = remediationPreview();
    if (kind === "another-package") preview.vulnerability!.packageName = "other";
    if (kind === "missing-range") preview.vulnerability!.affectedRanges = [];
    if (kind === "only-fixed-event") preview.vulnerability!.affectedRanges = [create(SecurityVersionRangeSchema, { fixed: "2.0.0" })];
    if (kind === "not-found") preview.found = false;
    renderWithProviders(<VulnerabilityRemediationDrawer {...props} preview={preview} />);
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Deny reviewed vulnerable range." } });
    expect(screen.getByRole("button", { name: "Apply denied range" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Apply denied range" }));
    expect(props.onApply).not.toHaveBeenCalled();
  });
});
