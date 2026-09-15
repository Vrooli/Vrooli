import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { authoredSample } from "../test-utils/readings";
import { ladderFixture, ladderReading } from "../test-utils/ladder";
import type { LadderRung } from "../lib/ladder";
import { LadderTile } from "./LadderTile";
import { NextRungReadout } from "./NextRungReadout";
import { ReachMapReadout } from "./ReachMapReadout";

const firstRung = (overrides: Partial<LadderRung>) => {
  const ladder = ladderFixture();
  const [head, ...rest] = ladder.rungs;
  return ladderFixture({ rungs: [{ ...(head as LadderRung), ...overrides }, ...rest] });
};

const unavailable = { coverage: "IN-REACH" as const, trust: "UNAVAILABLE" as const, value: null, observedAt: null, ladder: undefined };

describe("ladder views when there is nothing, or everything, to show", () => { // [REQ:CC-P1-016]
  it("each view says it has no schedule rather than drawing an empty frame", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading(unavailable)} />);
    expect(screen.getByText("No schedule to show.")).toBeInTheDocument();
  });
  it("the reach map says it has no schedule", () => {
    renderWithProviders(<ReachMapReadout reading={ladderReading(unavailable)} />);
    expect(screen.getByText("No schedule to show.")).toBeInTheDocument();
  });
  it("the reach map marks an authored schedule as illustrative", () => {
    const { container } = renderWithProviders(<ReachMapReadout reading={ladderReading({ ...unavailable, sample: { ...authoredSample(5), ladder: ladderFixture() } })} />);
    expect(container.firstElementChild).toHaveAttribute("data-provenance", "sample");
    expect(screen.getByText("illustrative")).toBeInTheDocument();
  });
  it("the strip tile shows a dash with no schedule and says so when everything shipped", () => {
    const { unmount } = renderWithProviders(<ul><LadderTile reading={ladderReading(unavailable)} /></ul>);
    expect(screen.getByText("—")).toBeInTheDocument();
    unmount();
    renderWithProviders(<ul><LadderTile reading={ladderReading({ ladder: ladderFixture({ nextRank: 0, unscheduled: [] }) })} /></ul>);
    expect(screen.getByText("All scheduled releases shipped")).toBeInTheDocument();
    expect(screen.getByText("4 of 5 still ideas")).toBeInTheDocument();
  });
  it("the strip tile keeps an authored schedule marked as a sample", () => {
    const { container } = renderWithProviders(<ul><LadderTile reading={ladderReading({ ...unavailable, sample: { ...authoredSample(5), ladder: ladderFixture() } })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-provenance", "sample");
  });
});

describe("Next Rung detail", () => { // [REQ:CC-P1-016]
  it("reports a closed readiness goal with its approved commit", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading({ ladder: firstRung({ readiness: { reported: true, goalClosed: true, approvedCommit: "abcdef1234567890" } }) })} />);
    expect(screen.getByText("readiness goal closed · abcdef12")).toBeInTheDocument();
  });
  it("reports an open readiness goal", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading({ ladder: firstRung({ readiness: { reported: true } }) })} />);
    expect(screen.getByText("readiness goal open")).toBeInTheDocument();
  });
  it("caps the blocker list and counts the rest", () => {
    const blockers = Array.from({ length: 7 }, (_, index) => ({ name: `work-${index}`, status: "IDEA", urgency: 1 }));
    renderWithProviders(<NextRungReadout reading={ladderReading({ ladder: firstRung({ blockers }) })} />);
    expect(screen.getByText("7 open")).toBeInTheDocument();
    expect(screen.getByText("+2 more")).toBeInTheDocument();
    expect(screen.queryByText("work-5")).toBeNull();
  });
  it("says when nothing is in the way and when a release opens nothing new", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading({ ladder: firstRung({ blockers: [], ramps: [], streams: [], audiences: [], goals: [], finishBar: undefined }) })} />);
    expect(screen.getByText("Nothing enabling is still open.")).toBeInTheDocument();
    expect(screen.getByText("Opens no new ramp or stream")).toBeInTheDocument();
    expect(screen.getByText("none linked")).toBeInTheDocument();
    expect(screen.getByText("Next release")).toBeInTheDocument();
  });
  it("names an unavailable goal source instead of hiding it", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading({ ladder: ladderFixture({ unavailable: [{ source: "swarm-manager.goals", reason: "connection refused" }] }) })} />);
    expect(screen.getByText("swarm-manager.goals unavailable · connection refused")).toBeInTheDocument();
  });
});
