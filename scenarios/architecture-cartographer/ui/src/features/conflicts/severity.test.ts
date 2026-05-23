import { describe, expect, it } from "vitest";

import { severityToLevel } from "./severity";
import { Severity } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

describe("severityToLevel", () => {
  it("maps every proto severity to a SeverityLevel", () => {
    expect(severityToLevel(Severity.UNSPECIFIED)).toBe("info");
    expect(severityToLevel(Severity.INFO)).toBe("info");
    expect(severityToLevel(Severity.WARN)).toBe("medium");
    expect(severityToLevel(Severity.ERROR)).toBe("high");
    expect(severityToLevel(Severity.BLOCKER)).toBe("critical");
  });
});
