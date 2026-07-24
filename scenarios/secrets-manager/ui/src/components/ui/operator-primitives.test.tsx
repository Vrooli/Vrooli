import { cleanup, fireEvent, screen } from "@testing-library/react";
import { AlertCircle } from "lucide-react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ConfirmDialog } from "./ConfirmDialog";
import { HelpDialog } from "./HelpDialog";
import { SecretsRow } from "./SecretsRow";
import { SeverityBadge } from "./SeverityBadge";
import { StatusTile } from "./StatusTile";
import { VulnerabilityItem } from "./VulnerabilityItem";

describe("operator-facing UI primitives", () => {
  afterEach(cleanup);

  it("keeps destructive confirmation hidden until requested and sends only explicit actions", () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const { rerender } = renderWithProviders(
      <ConfirmDialog open={false} title="Delete strategy" message="This cannot be undone." onConfirm={onConfirm} onCancel={onCancel} />
    );
    expect(screen.queryByText("Delete strategy")).not.toBeInTheDocument();

    rerender(
      <ConfirmDialog
        open
        title="Delete strategy"
        message="This cannot be undone."
        confirmLabel="Delete"
        variant="danger"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("opens help content and lets the operator dismiss it without triggering the dialog body", () => {
    renderWithProviders(
      <HelpDialog title="Deployment help">
        <p>Choose a strategy for every required secret.</p>
      </HelpDialog>
    );

    fireEvent.click(screen.getByRole("button", { name: "Help" }));
    expect(screen.getByText("Choose a strategy for every required secret.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close help dialog" }));
    expect(screen.queryByText("Choose a strategy for every required secret.")).not.toBeInTheDocument();
  });

  it("renders secret readiness, pluralization, and the workbench action for incomplete resources", () => {
    const onOpenResource = vi.fn();
    renderWithProviders(
      <SecretsRow
        status={{
          resource_name: "vault",
          secrets_total: 3,
          secrets_found: 1,
          secrets_missing: 2,
          secrets_optional: 0,
          health_status: "critical",
          last_checked: "2026-07-23T00:00:00Z"
        }}
        onOpenResource={onOpenResource}
      />
    );

    expect(screen.getByText("1/3 configured")).toBeInTheDocument();
    expect(screen.getByText("Missing 2 secrets")).toBeInTheDocument();
    expect(screen.getByText("33% ready")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open Workbench" }));
    expect(onOpenResource).toHaveBeenCalledWith("vault");
  });

  it("maps health vocabulary to readable status badges and renders vulnerability remediation context", () => {
    renderWithProviders(
      <>
        <SeverityBadge severity="healthy" />
        <SeverityBadge severity="unexpected" />
        <StatusTile icon={AlertCircle} label="Vault readiness" value="Ready" meta="No blockers" intent="good" />
        <VulnerabilityItem
          vuln={{
            id: "vuln-1",
            component_type: "resource",
            component_name: "vault",
            file_path: "resources/vault/resource.json",
            line_number: 18,
            severity: "high",
            type: "policy",
            title: "Missing policy",
            description: "The token needs a scoped policy.",
            recommendation: "Create a scoped policy.",
            can_auto_fix: false,
            discovered_at: "2026-07-23T00:00:00Z"
          }}
        />
      </>
    );

    expect(screen.getByText("healthy")).toBeInTheDocument();
    expect(screen.getByText("unexpected")).toBeInTheDocument();
    expect(screen.getByText("Vault readiness")).toBeInTheDocument();
    expect(screen.getByText("No blockers")).toBeInTheDocument();
    expect(screen.getByText("Recommendation: Create a scoped policy.")).toBeInTheDocument();
  });
});
