import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { searchDependencyRationale } from "../../api/dependencyRationale";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { DependencyRationalePanel } from "./DependencyRationalePanel";

vi.mock("../../api/dependencyRationale", () => ({ searchDependencyRationale: vi.fn() }));

describe("DependencyRationalePanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders distinct declaration fields with owner provenance and graph/package limits", async () => {
    vi.mocked(searchDependencyRationale).mockResolvedValue({
      query: "why runtime dependency", scenario: "web-console",
      results: [{ id: "dependency-rationale:web-console:notification-hub", title: "web-console → notification-hub", score: 0.8, snippet: "HTTP request for notifications", path: "web-console/.vrooli/service.json", consumer: "web-console", dependency: "notification-hub", type: "scenario", required: true, enabled: true, runtimeOnly: true, evidence: { description: "Send notifications.", purpose: "User notification delivery.", runtimeOnlyRationale: "Call over HTTP without importing source packages.", degradedBehavior: "Show notification failure and continue." }, source: { owner: "web-console", basis: "service_manifest.dependency_declaration", path: "web-console/.vrooli/service.json", fingerprint: "sha256:manifest", observedAt: "2026-11-24T00:00:00Z" } }],
      scan: { basis: "canonical_service_manifest", scenariosScanned: 1, scenariosFailed: 0, complete: true, runtimeStatusKnown: false }, degradations: [],
    });
    renderWithProviders(<DependencyRationalePanel scenario="web-console" />);
    fireEvent.change(screen.getByLabelText("Rationale query"), { target: { value: "why runtime dependency" } });
    fireEvent.click(screen.getByRole("button", { name: "Search this scenario" }));
    expect(await screen.findByText("web-console → notification-hub")).toBeInTheDocument();
    expect(screen.getByText("Send notifications.")).toBeInTheDocument();
    expect(screen.getByText("User notification delivery.")).toBeInTheDocument();
    expect(screen.getByText("Call over HTTP without importing source packages.")).toBeInTheDocument();
    expect(screen.getByText("Show notification failure and continue.")).toBeInTheDocument();
    expect(screen.getByText(/runtime status known false/)).toBeInTheDocument();
    expect(screen.getByText(/separate from detected graph edges and package approval rationale/)).toBeInTheDocument();
    expect(screen.getByText(/web-console\/\.vrooli\/service\.json · sha256:manifest/)).toBeInTheDocument();
    expect(searchDependencyRationale).toHaveBeenCalledWith("why runtime dependency", "web-console", 10);
  });

  it("preserves empty and partial evidence without implying dependency absence", async () => {
    vi.mocked(searchDependencyRationale).mockResolvedValue({
      query: "unmatched reason", scenario: "web-console", results: [],
      scan: { basis: "canonical_service_manifest", scenariosScanned: 1, scenariosFailed: 1, complete: false, runtimeStatusKnown: false },
      degradations: [{ scenario: "web-console", code: "manifest_unavailable", reason: "manifest could not be read" }],
    });
    renderWithProviders(<DependencyRationalePanel scenario="web-console" />);
    fireEvent.change(screen.getByLabelText("Rationale query"), { target: { value: "unmatched reason" } });
    fireEvent.click(screen.getByRole("button", { name: "Search this scenario" }));
    expect(await screen.findByText(/No declaration rationale matched this query in the selected scenario/)).toBeInTheDocument();
    expect(screen.getByText(/This is not evidence that the graph edge or dependency is absent/)).toBeInTheDocument();
    expect(screen.getByText(/manifest_unavailable: manifest could not be read/)).toBeInTheDocument();
    expect(screen.getByText(/1 manifest\(s\) could not be read/)).toBeInTheDocument();
  });
});
