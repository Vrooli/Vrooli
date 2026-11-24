import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import type { ContextCheckRow } from "./contextChecks";
import { InContextChecksPanel } from "./InContextChecks";

function row(
  id: string,
  verdict: ContextCheckRow["verdict"],
  evidence: ContextCheckRow["evidence"],
): ContextCheckRow {
  return {
    id,
    label: `${id} measurement`,
    detail: `${id} detail`,
    verdict,
    evidence,
    scored: true,
  };
}

describe("InContextChecksPanel", () => {
  it("explains that measurements are pending when no rows exist", () => {
    renderWithProviders(<InContextChecksPanel rows={[]} />, { withoutRouter: true });

    expect(screen.getByText(strings.workbench.component.ctxChecksEmpty)).toBeInTheDocument();
    expect(document.querySelector(".ic-checks")).not.toBeInTheDocument();
  });

  it("shows each verdict with its evidence provenance", () => {
    const rows = [
      row("layout", "pass", "measured"),
      row("compatibility", "fail", "declared"),
      row("contrast", "warn", "none"),
      row("touch", "unmeasured", "none"),
    ];

    renderWithProviders(<InContextChecksPanel rows={rows} />, { withoutRouter: true });

    const checks = [...document.querySelectorAll<HTMLElement>(".ic-check")];
    expect(checks.map((check) => check.className)).toEqual([
      "ic-check is-pass",
      "ic-check is-fail",
      "ic-check is-warn",
      "ic-check is-unmeasured",
    ]);
    expect(screen.getByText((content) => content === "layout measurement")).toBeInTheDocument();
    expect(screen.getByText((content) => content === "touch detail")).toBeInTheDocument();
    for (const verdict of ["PASS", "FAIL", "WARN", "N/M"]) {
      expect(screen.getByText(verdict)).toBeInTheDocument();
    }
    expect(screen.getByText(strings.workbench.component.ctxMeasured)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.component.ctxDeclared)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.component.ctxChecksNote)).toBeInTheDocument();
  });
});
