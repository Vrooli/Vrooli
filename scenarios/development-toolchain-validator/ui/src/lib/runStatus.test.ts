import { describe, expect, it } from "vitest";

import { ValidationRunStatus } from "../api/validationRun";
import { Verdict } from "../api/validationRecord";
import { runStatusLabelKey, runVerdictLabelKey } from "./runStatus";

describe("runStatus utilities", () => {
  it("maps every run status to a translation key", () => {
    expect(runStatusLabelKey(ValidationRunStatus.QUEUED)).toBe("runs.status.queued");
    expect(runStatusLabelKey(ValidationRunStatus.RUNNING)).toBe("runs.status.running");
    expect(runStatusLabelKey(ValidationRunStatus.EVALUATING)).toBe("runs.status.evaluating");
    expect(runStatusLabelKey(ValidationRunStatus.TERMINAL)).toBe("runs.status.terminal");
    expect(runStatusLabelKey(ValidationRunStatus.UNSPECIFIED)).toBe("runs.status.unknown");
    expect(runStatusLabelKey(999 as ValidationRunStatus)).toBe("runs.status.unknown");
  });

  it("maps every terminal verdict to a translation key", () => {
    expect(runVerdictLabelKey(Verdict.PASS)).toBe("runs.verdict.pass");
    expect(runVerdictLabelKey(Verdict.UNEXPECTED_MUTATION)).toBe("runs.verdict.unexpected");
    expect(runVerdictLabelKey(Verdict.RUN_FAILURE)).toBe("runs.verdict.runFailure");
    expect(runVerdictLabelKey(Verdict.TOOL_FAILURE)).toBe("runs.verdict.toolFailure");
    expect(runVerdictLabelKey(Verdict.UNSPECIFIED)).toBe("runs.verdict.pending");
    expect(runVerdictLabelKey(999 as Verdict)).toBe("runs.verdict.pending");
  });
});
