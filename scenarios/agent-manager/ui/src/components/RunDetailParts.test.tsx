import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, test } from "vitest";
import { RunSchema } from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";
import { RunStatus } from "../types";
import { getCostTotals, RunDetailsContent } from "./RunDetailParts";

function details(status: RunStatus, producer = "children", deadline = true) {
  const run = create(RunSchema, {
    id: "retained-parent", status,
    errorMsg: status === RunStatus.NEEDS_REVIEW ? "runner exited before terminal event" : "",
    awaitHandle: { producer, key: "retained-parent", ...(deadline ? { deadline: timestampFromDate(new Date("2026-10-04T09:54:14Z")) } : {}) },
  });
  return renderToStaticMarkup(<RunDetailsContent run={run} taskTitle="Existing delivery" profileName="Existing profile" durationMs={null} costTotals={getCostTotals([])} />);
}

describe("retained run wait evidence", () => {
  test("explains a child predicate separately from new feedback and its timer", () => {
    const html = details(RunStatus.PARKED);
    expect(html).toContain("children:retained-parent");
    expect(html).toContain("newly ended direct child");
    expect(html).toContain("already reported children do not wake it again");
    expect(html).toContain("New feedback alone does not satisfy this condition");
    expect(html).toContain("Timer deadline:");
    expect(html).not.toContain("at the latest");
    expect(html).not.toContain("not hung");
    expect(html).not.toContain("zero tokens");
  });
  test("does not project the child predicate onto other producers", () => {
    const html = details(RunStatus.PARKED, "test-genie", false);
    expect(html).toContain("test-genie:retained-parent");
    expect(html).not.toContain("newly ended direct child");
    expect(html).not.toContain("Timer deadline:");
  });
  test("an old handle does not label a crashed review run as parked", () => {
    const html = details(RunStatus.NEEDS_REVIEW);
    expect(html).toContain("runner exited before terminal event");
    expect(html).not.toContain("Parked — awaiting a result");
    expect(html).not.toContain("Timer deadline:");
  });
});
