import { describe, expect, it } from "vitest";

import { contextCommand, contextKindLabel, repeatLabel } from "./relevantContext";
import {
  RelevantContextItemSchema,
  RelevantContextKind,
  RelevantContextRepeatPolicy,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { create } from "@bufbuild/protobuf";

describe("relevantContext helpers", () => {
  it("formats context vocabulary consistently", () => {
    expect(contextKindLabel(RelevantContextKind.REQ_REF)).toBe("Requirement");
    expect(contextKindLabel(RelevantContextKind.REQ_REF, "lower")).toBe("requirement");
    expect(repeatLabel(RelevantContextRepeatPolicy.PHASE_ENTRY)).toBe("phase entry");
  });

  it("prefers argv over command text", () => {
    const item = create(RelevantContextItemSchema, {
      argv: ["prompt-manager", "skill", "read", "utils-unification"],
      command: "stale command",
    });
    expect(contextCommand(item)).toBe("prompt-manager skill read utils-unification");
  });
});
