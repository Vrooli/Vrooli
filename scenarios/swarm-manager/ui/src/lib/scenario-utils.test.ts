import { describe, expect, it } from "vitest";
import { scenariosFromGlobs } from "./scenario-utils";

describe("scenariosFromGlobs", () => {
  it("returns empty array for undefined", () => {
    expect(scenariosFromGlobs(undefined)).toEqual([]);
  });

  it("returns empty array for empty input", () => {
    expect(scenariosFromGlobs([])).toEqual([]);
  });

  it("skips non-scenario globs", () => {
    expect(scenariosFromGlobs(["packages/shared/**", "**/*.go"])).toEqual([]);
  });

  it("extracts single scenario name", () => {
    expect(scenariosFromGlobs(["scenarios/web-console/api/**"])).toEqual(["web-console"]);
  });

  it("deduplicates same scenario from multiple globs", () => {
    expect(
      scenariosFromGlobs(["scenarios/web-console/api/**", "scenarios/web-console/ui/**"]),
    ).toEqual(["web-console"]);
  });

  it("returns multiple distinct scenarios in order", () => {
    expect(
      scenariosFromGlobs(["scenarios/web-console/**", "scenarios/shared-ui/**"]),
    ).toEqual(["web-console", "shared-ui"]);
  });

  it("handles glob without sub-path", () => {
    expect(scenariosFromGlobs(["scenarios/web-console"])).toEqual(["web-console"]);
  });

  it("handles mixed scenario and non-scenario globs", () => {
    expect(
      scenariosFromGlobs([
        "packages/proto/**",
        "scenarios/web-console/api/**",
        "**/*.md",
        "scenarios/landing-page-business-suite/**",
      ]),
    ).toEqual(["web-console", "landing-page-business-suite"]);
  });

  it("skips bare 'scenarios/' prefix with no name", () => {
    expect(scenariosFromGlobs(["scenarios/"])).toEqual([]);
  });
});
