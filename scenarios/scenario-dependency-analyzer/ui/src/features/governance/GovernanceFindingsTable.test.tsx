import { create } from "@bufbuild/protobuf";
import { ApprovedDependencyFindingSchema, DependencyUsageGroupSchema } from "@vrooli/proto-types/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_pb";
import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { GovernanceDependenciesTable, GovernanceFindingsTable } from "./GovernanceFindingsTable";

describe("governance evidence selection", () => {
  it("keeps no-match states explicit without offering a decision on an absent dependency", () => {
    const onSelectDependency = vi.fn();
    renderWithProviders(<><GovernanceFindingsTable findings={[]} onSelectDependency={onSelectDependency} /><GovernanceDependenciesTable groups={[]} selectedKey={null} onSelect={vi.fn()} /></>);
    expect(screen.getByText("No findings match the current filters.")).toBeInTheDocument();
    expect(screen.getByText("No dependencies match the current filters.")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("selects the exact dependency from both evidence presentations and displays incomplete evidence honestly", () => {
    const onSelectDependency = vi.fn();
    const findings = [
      create(ApprovedDependencyFindingSchema, { id: "observed", ecosystem: "npm", packageName: "example", title: "Unreviewed dependency", severity: "warning", scenario: "alpha", observed: "^2.0.0", expected: "approved", description: "Review required.", findingClass: "unrecorded" }),
      create(ApprovedDependencyFindingSchema, { ecosystem: "go", packageName: "example.org/library", findingClass: "unrecorded" })
    ];
    renderWithProviders(<GovernanceFindingsTable findings={findings} onSelectDependency={onSelectDependency} />);
    const table = screen.getByRole("table", { name: "Governance findings" });
    expect(within(table).getByText("^2.0.0")).toBeInTheDocument();
    expect(within(table).getAllByText("-")).toHaveLength(3);
    fireEvent.click(within(table).getByRole("button", { name: "npm/example" }));
    expect(onSelectDependency).toHaveBeenLastCalledWith("npm", "example");
    fireEvent.click(screen.getByRole("button", { name: /warning Unreviewed dependency npm\/example Review required/ }));
    expect(onSelectDependency).toHaveBeenLastCalledWith("npm", "example");
    fireEvent.click(within(table).getByRole("button", { name: "go/example.org/library" }));
    expect(onSelectDependency).toHaveBeenLastCalledWith("go", "example.org/library");
  });

  it("preserves grouped dependency identities, severity fallbacks and selection across responsive views", () => {
    const groups = [
      create(DependencyUsageGroupSchema, { ecosystem: "npm", packageName: "first", state: "approved", scenarioCount: 2, usageCount: 3, findingCount: 1 }),
      create(DependencyUsageGroupSchema, { ecosystem: "npm", packageName: "second", highestSeverity: "warning" }),
      create(DependencyUsageGroupSchema, { ecosystem: "npm", packageName: "third" })
    ];
    const onSelect = vi.fn();
    renderWithProviders(<GovernanceDependenciesTable groups={groups} selectedKey="npm/first" onSelect={onSelect} />);
    const table = screen.getByRole("table", { name: "Dependency usage groups" });
    fireEvent.click(within(table).getByRole("button", { name: "npm/second" }));
    expect(onSelect).toHaveBeenLastCalledWith(groups[1]);
    expect(within(table).getByText("warning")).toBeInTheDocument();
    expect(within(table).getByText("info")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /npm\/first approved 2 scenarios, 3 usages, 1 findings/ }));
    expect(onSelect).toHaveBeenLastCalledWith(groups[0]);
  });
});
