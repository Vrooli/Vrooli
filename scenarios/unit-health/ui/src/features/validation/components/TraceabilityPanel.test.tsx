import { create } from "@bufbuild/protobuf";
import { cleanup } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { QualityReason, RequirementRegistration, RequirementExecutionState, RequirementApplicability, RequirementTraceabilityReportSchema } from "@vrooli/proto-types/unit-health/v1/validation/test_quality_pb";
import { renderWithProviders } from "../../../test-utils";
import { TraceabilityPanel } from "./TraceabilityPanel";

afterEach(cleanup);

it("separates registered links, skipped execution, and integration-only responsibility", () => {
  const report = create(RequirementTraceabilityReportSchema, { schemaVersion: "requirement-traceability/v1",
    requirements: [{ requirementId: "UH-CORE-010", applicability: RequirementApplicability.NOT_APPLICABLE }],
    links: [{ requirementId: "UH-CORE-001", registration: RequirementRegistration.REGISTERED, execution: RequirementExecutionState.SKIPPED,
      target: { file: "a.test.ts", testId: "parameter-2" }, runId: "current", reason: QualityReason.SKIPPED }],
  });
  const { container } = renderWithProviders(<TraceabilityPanel report={report} />);
  for (const text of ["UH-CORE-010 unit_responsibility=not_applicable", "UH-CORE-001 [registered] execution=skipped", "a.test.ts:parameter-2", "run=current"]) expect(container).toHaveTextContent(text);
  expect(container).not.toHaveTextContent("execution=passed");
});

it("keeps observed execution distinct from unavailable registry membership", () => {
  const report = create(RequirementTraceabilityReportSchema, { schemaVersion: "requirement-traceability/v1", unavailableReason: QualityReason.OWNER_UNAVAILABLE,
    links: [{ requirementId: "UH-CORE-001", registration: RequirementRegistration.REGISTERED, execution: RequirementExecutionState.PASSED }] });
  const { container, rerender } = renderWithProviders(<TraceabilityPanel report={report} />);
  expect(container).toHaveTextContent("[unknown] execution=passed");
  expect(container).toHaveTextContent("registry=unknown");
  report.schemaVersion = "future";
  rerender(<TraceabilityPanel report={report} />);
  expect(container).toHaveTextContent("[unknown] execution=unknown");
  expect(container).not.toHaveTextContent("execution=passed");
  report.schemaVersion = "requirement-traceability/v1";
  const link = report.links[0];
  if (!link) throw new Error("fixture must contain a link");
  link.execution = 999 as RequirementExecutionState;
  rerender(<TraceabilityPanel report={report} />);
  expect(container).toHaveTextContent("execution=unknown");
});

it("does not treat missing historical reports as an empty verified registry", () => {
  const { container } = renderWithProviders(<TraceabilityPanel />);
  expect(container).toHaveTextContent("traceabilityUnknown");
  expect(container).not.toHaveTextContent("registry=available");
});
