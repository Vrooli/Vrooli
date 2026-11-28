import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { DeploymentAnalysisReport } from "../../types";
import { renderWithProviders } from "../../test-utils";
import { DeploymentInsightsPanel } from "./DeploymentInsightsPanel";

const emptyReport = (): DeploymentAnalysisReport => ({ scenario: "alpha", report_version: 1, generated_at: "2026-10-16T12:00:00Z", dependencies: [], aggregates: {} });

describe("deployment packaging evidence", () => {
  it("makes missing readiness, bundle and dependency evidence explicit", () => {
    renderWithProviders(<DeploymentInsightsPanel report={emptyReport()} />);
    expect(screen.getByText("No tier metadata discovered yet.")).toBeInTheDocument();
    expect(screen.getByText("No bundle files recorded.")).toBeInTheDocument();
    expect(screen.getByText("No dependencies recorded.")).toBeInTheDocument();
    expect(screen.getByText("No blocking dependencies detected.")).toBeInTheDocument();
    expect(screen.getByText("vrooli scenario start deployment-manager")).toBeInTheDocument();
  });

  it("separates absent bundle files from present files and distinguishes dependency owners", () => {
    const report = emptyReport();
    report.bundle_manifest = { scenario: "alpha", generated_at: report.generated_at, dependencies: [], files: [{ path: "api/server", type: "binary", exists: true }, { path: "ui/dist/index.html", type: "ui", exists: false }] };
    report.dependencies = [{ name: "postgres", type: "resource" }, { name: "account-service", type: "scenario" }];
    renderWithProviders(<DeploymentInsightsPanel report={report} />);
    const missing = screen.getByText("ui: ui/dist/index.html").closest("li");
    const present = screen.getByText("binary: api/server").closest("li");
    if (!missing || !present) throw new Error("Expected bundle evidence rows");
    expect(within(missing).getByText("missing")).toBeInTheDocument();
    expect(within(present).getByText("present")).toBeInTheDocument();
    expect(screen.getByText("Resources")).toBeInTheDocument();
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.getByText("Scenarios")).toBeInTheDocument();
    expect(screen.getByText("account-service")).toBeInTheDocument();
  });

  it("shows tier-specific blockers and requirements, deduplicating cross-tier blocker summaries", () => {
    const report = emptyReport();
    report.aggregates = {
      mobile: { fitness_score: 0.25, dependency_count: 2, blocking_dependencies: ["postgres", "ollama"], estimated_requirements: { ram_mb: 512.55, disk_mb: 1024, cpu_cores: 0.25 } },
      desktop: { fitness_score: 0.75, dependency_count: 2, blocking_dependencies: ["postgres"], estimated_requirements: { ram_mb: 1024, disk_mb: 2048, cpu_cores: 1 } },
      local_dev: { fitness_score: 1, dependency_count: 2, estimated_requirements: { ram_mb: 1024, disk_mb: 2048, cpu_cores: 1 } }
    };
    renderWithProviders(<DeploymentInsightsPanel report={report} />);
    expect(screen.getByText("25%")).toBeInTheDocument();
    expect(screen.getByText("75%")).toBeInTheDocument();
    expect(screen.getByText("100%")).toBeInTheDocument();
    expect(screen.getByText(/RAM 512.6 MB · Disk 1024 MB · CPU 0.3 cores/)).toBeInTheDocument();
    expect(screen.getByText(/Blockers: none/)).toBeInTheDocument();
    const blockers = screen.getByText("Blockers", { exact: true }).parentElement;
    if (!blockers) throw new Error("Expected blocker summary");
    expect(within(blockers).getAllByText("postgres", { exact: true })).toHaveLength(1);
    expect(within(blockers).getByText("ollama", { exact: true })).toBeInTheDocument();
  });
});
