import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { PhaseExecutionResult } from "../../lib/api";
import { PhaseResultCard } from "./PhaseResultCard";

function evidence(context: Record<string, unknown>): string {
  return btoa(unescape(encodeURIComponent(JSON.stringify(context))));
}

describe("PhaseResultCard experience evidence", () => {
  it("renders ranked phone and desktop findings with identity, route, next action, and screenshot refs", () => {
    const phase: PhaseExecutionResult = {
      name: "experience",
      status: "failed",
      durationSeconds: 3,
      logPath: "",
      findings: [
        {
          stableId: "afid:desktop-type",
          code: "vision_ux_typography_hierarchy",
          severity: "WARNING",
          class: "heuristic",
          phase: "experience",
          gating: false,
          message: "Typography hierarchy is unclear on desktop",
          evidence: [{ kind: "ui_health_result.v1", locator: "/private/evidence/desktop.png", payload: evidence({ rank: 2, route: "/reports", viewport: "desktop (1440x900)", identity: "vision_ux_typography_hierarchy", nextAction: "Strengthen the page title hierarchy.", screenshotRef: "/private/evidence/desktop.png" }) }],
        },
        {
          stableId: "afid:phone-nav",
          code: "vision_ux_navigation_crowding",
          severity: "WARNING",
          class: "heuristic",
          phase: "experience",
          gating: false,
          message: "Navigation is crowded on the phone layout",
          evidence: [{ kind: "ui_health_result.v1", locator: "/private/evidence/phone.png", payload: evidence({ rank: 1, route: "/dashboard", viewport: "phone (390x844)", identity: "vision_ux_navigation_crowding", nextAction: "Reduce the visible navigation items.", screenshotRef: "/private/evidence/phone.png" }) }],
        },
      ],
    };

    render(<PhaseResultCard phase={phase} />);
    const cards = screen.getAllByTestId("experience-finding");
    expect(cards).toHaveLength(2);
    const first = cards[0]!;
    const second = cards[1]!;
    expect(within(first).getByText("#1")).toBeTruthy();
    expect(within(first).getByText(/vision_ux_navigation_crowding · \/dashboard · phone \(390x844\)/)).toBeTruthy();
    expect(within(first).getByText("Next: Reduce the visible navigation items.")).toBeTruthy();
    expect(within(first).getByText(/\/private\/evidence\/phone.png/)).toBeTruthy();
    expect(within(second).getByText("#2")).toBeTruthy();
    expect(within(second).getByText(/vision_ux_typography_hierarchy · \/reports · desktop \(1440x900\)/)).toBeTruthy();
    expect(within(second).getByText("Next: Strengthen the page title hierarchy.")).toBeTruthy();
    expect(within(second).getByText(/\/private\/evidence\/desktop.png/)).toBeTruthy();
  });

  it("omits opaque, malformed, incomplete, and generic findings from the ranked UX list", () => {
    const phase: PhaseExecutionResult = {
      name: "experience",
      status: "passed",
      durationSeconds: 1,
      logPath: "",
      findings: [
        { stableId: "afid:opaque", code: "unknown", severity: "INFO", class: "deterministic", phase: "experience", gating: false, evidence: [{ kind: "ui_health_result.v1", payload: "%%%" }] },
        { stableId: "afid:incomplete", code: "vision_ux_style", severity: "WARNING", class: "heuristic", phase: "experience", gating: false, evidence: [{ kind: "ui_health_result.v1", payload: evidence({ route: "/", viewport: "phone" }) }] },
        { stableId: "afid:static", code: "manifest_contract_error", severity: "ERROR", class: "deterministic", phase: "experience", gating: true, message: "Manifest is missing a required slot." },
      ],
    };
    render(<PhaseResultCard phase={phase} />);
    expect(screen.queryByTestId("experience-finding")).toBeNull();
    expect(screen.queryByText("#1")).toBeNull();
    expect(screen.getByTestId("experience-evidence-status").textContent).toContain("does not establish that the UI has no visual defects");
  });

  it("marks a skipped vision critique as unavailable without changing the phase gate", () => {
    const phase: PhaseExecutionResult = {
      name: "experience",
      status: "passed",
      durationSeconds: 1,
      logPath: "",
      findings: [{ stableId: "vision-skipped", code: "vision_ux_judge_skipped", severity: "INFO", class: "deterministic", phase: "experience", gating: false, message: "Local vision unavailable" }],
    };
    render(<PhaseResultCard phase={phase} />);
    expect(screen.getByTestId("experience-evidence-status").textContent).toContain("Local visual critique was skipped");
    expect(screen.getByText("passed")).toBeTruthy();
  });

  it("shows partial-coverage status when some routes have findings and another critique skipped", () => {
    const phase: PhaseExecutionResult = {
      name: "experience",
      status: "passed",
      durationSeconds: 2,
      logPath: "",
      findings: [
        { stableId: "vision-skipped", code: "vision_ux_judge_skipped", severity: "INFO", class: "deterministic", phase: "experience", gating: false },
        { stableId: "vision-result", code: "vision_ux_spacing_layout", severity: "WARNING", class: "heuristic", phase: "experience", gating: false,
          evidence: [{ kind: "ui_health_result.v1", payload: evidence({ rank: 1, route: "/", viewport: "phone", identity: "vision_ux_spacing_layout", nextAction: "Review the spacing.", screenshotRef: "/private/capture.png" }) }] },
      ],
    };
    render(<PhaseResultCard phase={phase} />);
    expect(screen.getByTestId("experience-evidence-status").textContent).toContain("findings below are incomplete");
    expect(screen.getAllByTestId("experience-finding")).toHaveLength(1);
    expect(screen.getByText("passed")).toBeTruthy();
  });

  it("previews a sanitized loopback screenshot reference", () => {
    const phase: PhaseExecutionResult = {
      name: "experience",
      status: "passed",
      durationSeconds: 1,
      logPath: "",
      findings: [{
        stableId: "afid:loopback",
        code: "vision_ux_spacing_layout",
        severity: "WARNING",
        class: "heuristic",
        phase: "experience",
        gating: false,
        evidence: [{ kind: "ui_health_result.v1", payload: evidence({ rank: 1, route: "/", viewport: "phone", identity: "vision_ux_spacing_layout", nextAction: "Review spacing.", screenshotRef: "http://127.0.0.1:4312/evidence/phone.png" }) }],
      }],
    };

    render(<PhaseResultCard phase={phase} />);
    const preview = screen.getByTestId("experience-screenshot-preview");
    expect(preview.getAttribute("src")).toBe("http://127.0.0.1:4312/evidence/phone.png");
    expect(preview.getAttribute("referrerPolicy")).toBe("no-referrer");
    expect(screen.getByRole("link", { name: "/evidence/phone.png" }).getAttribute("href")).toBe("http://127.0.0.1:4312/evidence/phone.png");
  });

  it("never fetches malformed, external, credentialed, query-bearing, or private-scheme screenshot refs", () => {
    const refs = [
      "https://screens.example.com/capture.png",
      "http://user:secret@localhost/capture.png",
      "http://localhost/capture.png?token=secret",
      "http://127.0.0.1/capture.png#private",
      "data:image/png;base64,AAAA",
      "file:///tmp/private.png",
      "not a url",
    ];
    const phase: PhaseExecutionResult = {
      name: "experience",
      status: "passed",
      durationSeconds: 1,
      logPath: "",
      findings: refs.map((screenshotRef, index) => ({
        stableId: `afid:unsafe-${index}`,
        code: "vision_ux_visual_polish",
        severity: "WARNING",
        class: "heuristic",
        phase: "experience",
        gating: false,
        evidence: [{ kind: "ui_health_result.v1", payload: evidence({ rank: index + 1, route: "/", viewport: "desktop", identity: "vision_ux_visual_polish", nextAction: "Review the page.", screenshotRef }) }],
      })),
    };

    render(<PhaseResultCard phase={phase} />);
    const cards = screen.getAllByTestId("experience-finding");
    expect(cards).toHaveLength(refs.length);
    for (const card of cards) {
      expect(card.querySelector("img")).toBeNull();
      expect(card.querySelector("a")).toBeNull();
      expect(card.textContent).toContain("Screenshot evidence:");
    }
    expect(document.querySelectorAll("img")).toHaveLength(0);
  });
});
