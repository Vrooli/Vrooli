import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import type { ContextCheckRow } from "./contextChecks";
import { InContextChecksPanel } from "./InContextChecks";

function row(
  id: string,
  verdict: ContextCheckRow["verdict"],
  group: ContextCheckRow["group"] = "fit",
): ContextCheckRow {
  return {
    id,
    label: `${id} measurement`,
    detail: `${id} detail`,
    verdict,
    scored: group !== "info",
    group,
  };
}

describe("InContextChecksPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("explains that measurements are pending when no rows exist", () => {
    renderWithProviders(<InContextChecksPanel rows={[]} />, { withoutRouter: true });

    expect(screen.getByTestId("checks-empty")).toBeInTheDocument();
    expect(document.querySelector(".ic-checks")).not.toBeInTheDocument();
  });

  it("keeps component fit visible and puts page, evidence and human behind one disclosure", () => {
    const rows = [
      row("slot", "pass", "fit"),
      row("overflow", "fail", "page"),
      row("states", "warn", "evidence"),
      row("squint", "unmeasured", "human"),
    ];
    renderWithProviders(<InContextChecksPanel rows={rows} />, { withoutRouter: true });

    expect(screen.getByRole("heading", { name: "Component" })).toBeInTheDocument();
    const disclosure = document.querySelector(".ic-secondary");
    expect(disclosure).not.toHaveAttribute("open");

    fireEvent.click(screen.getByText(/Page & evidence/));
    expect(disclosure).toHaveAttribute("open");
    expect(screen.getByRole("heading", { name: "Page" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Evidence" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Human" })).toBeInTheDocument();
  });

  it("shows each verdict without repeating evidence provenance", () => {
    const rows = [
      row("layout", "pass"),
      row("compatibility", "fail"),
      row("contrast", "warn"),
      row("touch", "unmeasured"),
    ];

    renderWithProviders(<InContextChecksPanel rows={rows} />, { withoutRouter: true });

    const checks = [...document.querySelectorAll<HTMLElement>(".ic-check")];
    expect(checks.map((check) => check.className)).toEqual([
      "ic-check is-pass",
      "ic-check is-fail",
      "ic-check is-warn",
      "ic-check is-unmeasured",
    ]);
    const labels = [...document.querySelectorAll(".ic-check-verdict")].map(
      (entry) => entry.textContent,
    );
    expect(labels).toEqual(["PASS", "FAIL", "WARN", "N/M"]);
    expect(document.querySelector(".ic-check-verdict em")).toBeNull();
    expect(screen.getByTestId("checks-note")).toHaveTextContent("N/M is not a pass.");
  });

  it("renders type as a footnote, not a scored row", () => {
    const rows = [row("size", "pass"), row("type", "unmeasured", "info")];
    renderWithProviders(<InContextChecksPanel rows={rows} />, { withoutRouter: true });

    expect(document.querySelectorAll(".ic-check")).toHaveLength(1);
    expect(document.querySelector(".ic-type-note")).toHaveTextContent("type detail");
  });

  it("uses the compact fixture variant without a disclosure", () => {
    const rows = [row("size", "pass"), row("touch", "unmeasured")];
    renderWithProviders(
      <InContextChecksPanel
        rows={rows}
        variant="fixture"
        evidenceNote="2 capture records"
        onRunEvidence={() => undefined}
        onOpenContext={() => undefined}
      />,
      { withoutRouter: true },
    );

    expect(document.querySelector(".ic-secondary")).not.toBeInTheDocument();
    expect(document.querySelector(".ic-note")).toHaveTextContent("Evidence: 2 capture records");
    expect(screen.getByText(/Page health is not checked here/)).toBeInTheDocument();
  });
});
