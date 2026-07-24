import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { SecurityTables } from "./SecurityTables";
import { StatusGrid } from "./StatusGrid";

describe("StatusGrid", () => {
  afterEach(cleanup);

  it("summarizes healthy dependencies, Vault coverage, and scan risk", () => {
    renderWithProviders(
      <StatusGrid
        healthData={{ status: "healthy", service: "secrets-manager", version: "2.0", timestamp: "2026-07-23T00:00:00Z", dependencies: { database: { connected: true, latency_ms: 4 } } }}
        vaultData={{ total_resources: 2, configured_resources: 1, missing_secrets: [{ resource_name: "vault", secret_name: "TOKEN", secret_path: "secret/token", required: true, description: "token" }], resource_statuses: [], last_updated: "2026-07-23T00:00:00Z" }}
        complianceData={{ overall_score: 90, vault_secrets_health: 80, vulnerability_summary: {}, remediation_progress: { configured_components: 1, critical_issues: 0, high_issues: 0, medium_issues: 0, low_issues: 0, security_score: 90, vault_secrets_health: 80, overall_compliance: 90 }, total_resources: 2, configured_resources: 1, configured_components: 1, total_components: 2, total_vulnerabilities: 0, last_updated: "2026-07-23T00:00:00Z" }}
        vulnerabilityData={{ vulnerabilities: [], total_count: 0, scan_id: "scan-1", scan_duration: 15, risk_score: 71 }}
        isHealthLoading={false}
        isVaultLoading={false}
        isComplianceLoading={false}
        isVulnerabilityLoading={false}
      />
    );
    expect(screen.getByText("API Terminal")).toBeInTheDocument();
    expect(screen.getByText("HEALTHY")).toBeInTheDocument();
    expect(screen.getByText("CONNECTED")).toBeInTheDocument();
    expect(screen.getByText("1/2")).toBeInTheDocument();
    expect(screen.getByText("1 missing")).toBeInTheDocument();
    expect(screen.getByText("15 ms")).toBeInTheDocument();
  });

  it("uses safe defaults and loading tiles while service data is unavailable", () => {
    renderWithProviders(
      <StatusGrid
        isHealthLoading={false}
        isVaultLoading={false}
        isComplianceLoading
        isVulnerabilityLoading={false}
      />
    );
    expect(screen.getByText("UNKNOWN")).toBeInTheDocument();
    expect(screen.getByText("DISCONNECTED")).toBeInTheDocument();
    expect(screen.getByText("0/0")).toBeInTheDocument();
    expect(screen.getByText("0 missing")).toBeInTheDocument();
    expect(document.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
  });
});

describe("SecurityTables", () => {
  afterEach(cleanup);

  it("renders findings, opens incomplete resources, and applies all filters", () => {
    const onOpenResource = vi.fn();
    const onComponentTypeChange = vi.fn();
    const onComponentFilterChange = vi.fn();
    const onSeverityFilterChange = vi.fn();
    renderWithProviders(
      <SecurityTables
        resourceStatuses={[{ resource_name: "vault", secrets_total: 2, secrets_found: 1, secrets_missing: 1, secrets_optional: 0, health_status: "degraded", last_checked: "now" }]}
        vulnerabilities={[{ id: "v-1", component_type: "resource", component_name: "vault", file_path: "resource.json", line_number: 7, severity: "high", type: "configuration", title: "Missing token", description: "A required token is absent", recommendation: "Configure the token", can_auto_fix: false, discovered_at: "now" }]}
        isVaultLoading={false}
        isVulnerabilityLoading={false}
        componentType=""
        componentFilter=""
        severityFilter=""
        componentOptions={["vault"]}
        scanId="scan-1"
        riskScore={72}
        scanDuration={15}
        onOpenResource={onOpenResource}
        onComponentTypeChange={onComponentTypeChange}
        onComponentFilterChange={onComponentFilterChange}
        onSeverityFilterChange={onSeverityFilterChange}
      />
    );
    expect(screen.getByText("1/2 configured")).toBeInTheDocument();
    expect(screen.getByText("Missing token")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open Workbench" }));
    fireEvent.change(screen.getByLabelText("Component type"), { target: { value: "resource" } });
    fireEvent.change(screen.getByLabelText("Component"), { target: { value: "vault" } });
    fireEvent.change(screen.getByLabelText("Severity"), { target: { value: "high" } });
    expect(onOpenResource).toHaveBeenCalledWith("vault");
    expect(onComponentTypeChange).toHaveBeenCalledWith("resource");
    expect(onComponentFilterChange).toHaveBeenCalledWith("vault");
    expect(onSeverityFilterChange).toHaveBeenCalledWith("high");
  });

  it("shows loading placeholders and empty-result states", () => {
    const { rerender } = renderWithProviders(
      <SecurityTables
        resourceStatuses={[]}
        vulnerabilities={[]}
        isVaultLoading
        isVulnerabilityLoading
        componentType=""
        componentFilter=""
        severityFilter=""
        componentOptions={[]}
        onComponentTypeChange={() => {}}
        onComponentFilterChange={() => {}}
        onSeverityFilterChange={() => {}}
      />
    );
    expect(screen.queryByText("No resource scan results available yet.")).not.toBeInTheDocument();
    rerender(
      <SecurityTables
        resourceStatuses={[]}
        vulnerabilities={[]}
        isVaultLoading={false}
        isVulnerabilityLoading={false}
        componentType=""
        componentFilter=""
        severityFilter=""
        componentOptions={[]}
        onComponentTypeChange={() => {}}
        onComponentFilterChange={() => {}}
        onSeverityFilterChange={() => {}}
      />
    );
    expect(screen.getByText("No resource scan results available yet.")).toBeInTheDocument();
    expect(screen.getByText("No vulnerabilities found for the selected filters.")).toBeInTheDocument();
  });
});
