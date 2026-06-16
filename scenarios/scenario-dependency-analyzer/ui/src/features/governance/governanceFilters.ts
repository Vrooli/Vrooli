import type {
  ApprovedDependencyFinding,
  ApprovedDependencyRecord,
  DependencyUsageGroup
} from "../../api/governance";
import type { GovernanceFilters } from "./governanceTypes";

const includesText = (value: string, query: string) =>
  value.toLowerCase().includes(query.toLowerCase());

export function matchesFinding(finding: ApprovedDependencyFinding, filters: GovernanceFilters): boolean {
  const query = filters.query.trim();
  if (filters.ecosystem !== "all" && finding.ecosystem !== filters.ecosystem) return false;
  if (filters.severity !== "all" && finding.severity !== filters.severity) return false;
  if (filters.scenario !== "all" && finding.scenario !== filters.scenario) return false;
  if (filters.state !== "all" && finding.findingClass !== filters.state && finding.expected !== filters.state) return false;
  if (!query) return true;
  return [
    finding.packageName,
    finding.ecosystem,
    finding.title,
    finding.description,
    finding.remediation,
    finding.scenario,
    finding.filePath,
    finding.findingClass
  ].some((value) => includesText(value, query));
}

export function matchesUsageGroup(group: DependencyUsageGroup, filters: GovernanceFilters): boolean {
  const query = filters.query.trim();
  if (filters.ecosystem !== "all" && group.ecosystem !== filters.ecosystem) return false;
  if (filters.severity !== "all" && group.highestSeverity !== filters.severity) return false;
  if (filters.scenario !== "all" && !group.scenarios.includes(filters.scenario)) return false;
  if (filters.state !== "all" && group.state !== filters.state) return false;
  if (!query) return true;
  return [
    group.packageName,
    group.ecosystem,
    group.state,
    group.highestSeverity,
    ...group.scenarios
  ].some((value) => includesText(value, query));
}

export function matchesRecord(record: ApprovedDependencyRecord, filters: GovernanceFilters): boolean {
  const query = filters.query.trim();
  if (filters.ecosystem !== "all" && record.ecosystem !== filters.ecosystem) return false;
  if (filters.state !== "all" && record.state !== filters.state) return false;
  if (filters.scenario !== "all" && record.allowedScenarios.length > 0 && !record.allowedScenarios.includes(filters.scenario)) {
    return false;
  }
  if (!query) return true;
  return [
    record.packageName,
    record.ecosystem,
    record.state,
    record.versionRange,
    record.rationale,
    record.replacement,
    ...record.keywords,
    ...record.useCases,
    ...record.exampleScenarios
  ].some((value) => includesText(value, query));
}
