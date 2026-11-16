import { describe, expect, it } from "vitest";
import { workflowExecutionLink } from "./workflowExecutionLink";

describe("workflowExecutionLink", () => {
  it("links the exact owner workflow ID on the current browser host", () => {
    expect(workflowExecutionLink("workflow / 1", "localhost", "https:")).toBe("https://localhost:21238/workflows?executionId=workflow+%2F+1");
  });

  it("does not produce a link without an owner execution ID", () => {
    expect(workflowExecutionLink("  ", "localhost")).toBe("");
  });
});
