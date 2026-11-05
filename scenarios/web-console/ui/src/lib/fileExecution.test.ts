import { describe, expect, it } from "vitest";
import { scriptRunPlan } from "./fileExecution";

describe("scriptRunPlan", () => {
  it("detects common script types and uses the script directory", () => {
    expect(scriptRunPlan("/work/tools/check.py") ).toMatchObject({
      command: "python3 '/work/tools/check.py'",
      workingDir: "/work/tools",
      interpreter: "python3",
      risky: false,
    });
  });

  it("prefers a shebang interpreter", () => {
    expect(scriptRunPlan("/work/run", "#!/usr/bin/env bash\necho ok")).toMatchObject({
      command: "bash '/work/run'",
      workingDir: "/work",
    });
  });

  it("marks commands with system-changing operations as risky", () => {
    expect(scriptRunPlan("/work/install.sh", "#!/bin/bash\nsudo apt install jq")).toMatchObject({ risky: true });
  });

  it("returns null for files without a recognized runner", () => {
    expect(scriptRunPlan("/work/README.md", "plain text")).toBeNull();
  });
});

