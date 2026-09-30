import { create } from "@bufbuild/protobuf";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it } from "vitest";
import { QualityCheckStatus, QualityReason, QualityEvidenceKind, QualityEnforcement, QualitySeverity, TestQualityReportSchema } from "@vrooli/proto-types/unit-health/v1/validation/test_quality_pb";
import { renderWithProviders } from "../../../test-utils";
import { QualityPanel } from "./QualityPanel";

afterEach(cleanup);

it("keeps all four states distinct in a partial collection", () => {
  const report = create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1", totalResults: 4n,
    coverage: [{ ruleId: "rule", supportProfile: "profile", discovered: 4n, assessed: 2n, unknown: 1n, notApplicable: 1n }],
    collectionLimitations: [{ reason: QualityReason.OWNER_UNAVAILABLE, guidance: "Check missing collector." }],
    results: [QualityCheckStatus.CHECKED_CLEAN, QualityCheckStatus.VIOLATION, QualityCheckStatus.UNKNOWN, QualityCheckStatus.NOT_APPLICABLE].map((status) => ({ status })) });
  const { container } = renderWithProviders(<QualityPanel report={report} />);
  for (const text of ["[checked_clean]", "[violation]", "[unknown]", "[not_applicable]", "assessed=2/discovered=4 unknown=1 not_applicable=1", "Check missing collector."]) expect(container).toHaveTextContent(text);
});

it("preserves an observed violation while disclosing incomplete collection", () => {
  const report = create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1", totalResults: 1n,
    collectionLimitations: [{ reason: QualityReason.OWNER_UNAVAILABLE, guidance: "Check the missing collector." }],
    coverage: [{ ruleId: "assertion", supportProfile: "vitest", discovered: 1n, assessed: 1n }],
    results: [{ ruleId: "assertion", status: QualityCheckStatus.VIOLATION, enforcement: QualityEnforcement.BLOCKING }] });
  const { container } = renderWithProviders(<QualityPanel report={report} />);
  expect(screen.getByText("Check the missing collector.")).toBeVisible();
  for (const text of ["partialCollectionLimit", "[violation]", "BLOCKING", "assessed=1/discovered=1"]) expect(container).toHaveTextContent(text);
});

it("shows the provider recovery action beside unknown evidence", () => {
  const report = create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1", unavailableReason: QualityReason.OWNER_UNAVAILABLE, reasonGuidance: "Check the owner health.", results: [{ reasonGuidance: "Inspect native skip rationale.", status: QualityCheckStatus.UNKNOWN }] });
  const { container } = renderWithProviders(<QualityPanel report={report} />);
  expect(screen.getByText("Check the owner health.")).toBeVisible();
  expect(container).toHaveTextContent("Inspect native skip rationale.");
});

it("retains partial denominators, stable identities, and separate evidence/enforcement", () => {
  const report = create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1", totalResults: 2n,
    coverage: [{ ruleId: "assertion", supportProfile: "vitest", discovered: 2n, assessed: 1n, unknown: 1n }],
    results: [{ ruleId: "assertion", status: QualityCheckStatus.CHECKED_CLEAN, evidenceKind: QualityEvidenceKind.STATIC,
      severity: QualitySeverity.ERROR, enforcement: QualityEnforcement.ADVISORY, target: { file: "a.test.ts", testId: "case-1" }, location: { line: 7 } },
    { ruleId: "assertion", status: QualityCheckStatus.UNKNOWN, reason: QualityReason.SKIPPED, target: { file: "a.test.ts", testId: "case-2" } }],
  });
  const { container } = renderWithProviders(<QualityPanel report={report} />);
  for (const text of ["assessed=1/discovered=2", "unknown=1", "a.test.ts:7 case-1", "[unknown]", "SKIPPED", "STATIC", "ERROR", "ADVISORY"]) expect(container).toHaveTextContent(text);
  expect(container.querySelectorAll("details")).toHaveLength(2);
});

it("does not turn absent historical assessment or inconsistent counts into clean evidence", () => {
  const { rerender, container } = renderWithProviders(<QualityPanel />);
  expect(container).not.toHaveTextContent("[checked_clean]");
  expect(container.textContent).toContain("qualityUnknown");
  rerender(<QualityPanel report={create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1",
    coverage: [{ ruleId: "rule", supportProfile: "profile", discovered: 1n, assessed: 2n }] })} />);
  expect(screen.getByText(/inconsistent denominator/)).toBeInTheDocument();
});

it("retains locations but marks unavailable or future states unknown", () => {
  const report = create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1", unavailableReason: QualityReason.OWNER_UNAVAILABLE,
    results: [{ ruleId: "rule", status: QualityCheckStatus.CHECKED_CLEAN, target: { file: "retained.test.ts", testId: "old" } }] });
  const { container } = renderWithProviders(<QualityPanel report={report} />);
  expect(container).toHaveTextContent("retained.test.ts");
  expect(container).toHaveTextContent("[unknown]");
  expect(container).not.toHaveTextContent("[checked_clean]");
});

it("bounds the initial result list and exposes retained additional results", async () => {
  const rows = Array.from({ length: 51 }, (_, i) => ({ ruleId: "rule", target: { file: "a.test.ts", testId: `case-${i}` }, status: QualityCheckStatus.UNKNOWN }));
  const { container } = renderWithProviders(<QualityPanel report={create(TestQualityReportSchema, { schemaVersion: "test-quality/v1", catalogVersion: "1", totalResults: 51n, results: rows })} />);
  expect(container.querySelectorAll("details")).toHaveLength(50);
  expect(screen.queryByText(/case-50/)).not.toBeInTheDocument();
  await userEvent.setup().click(screen.getByRole("button"));
  expect(container.querySelectorAll("details")).toHaveLength(51);
  expect(screen.getByText(/case-50/)).toBeInTheDocument();
});
