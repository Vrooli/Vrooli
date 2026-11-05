import { create } from "@bufbuild/protobuf";
import { cleanup } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { EvidenceStagesSchema } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";
import { renderWithProviders } from "../../../test-utils";
import { EvidenceStagesPanel } from "./EvidenceStagesPanel";

afterEach(cleanup);

it("separates static-only assessment from execution and review", () => {
  const { container } = renderWithProviders(<EvidenceStagesPanel stages={create(EvidenceStagesSchema, { configured: "observed", analyzed: "partial", executed: "not_requested", reviewed: "not_supplied", sourceRunId: "original" })} />);
  for (const text of ["configuredobserved", "analyzedpartial", "executednot_requested", "reviewednot_supplied", "source_run=original"]) expect(container).toHaveTextContent(text);
  expect(container).not.toHaveTextContent("executedpassed");
});

it("keeps missing and future stages unknown and cached execution distinct", () => {
  const { container, rerender } = renderWithProviders(<EvidenceStagesPanel />);
  expect(container.querySelectorAll("dd")).toHaveLength(4);
  for (const value of container.querySelectorAll("dd")) expect(value).toHaveTextContent("unknown");
  rerender(<EvidenceStagesPanel stages={create(EvidenceStagesSchema, { executed: "cached", reviewed: "passed", sourceRunId: "old" })} />);
  expect(container).toHaveTextContent("executedcached");
  expect(container).toHaveTextContent("reviewedunknown");
  expect(container).toHaveTextContent("source_run=old");
});
