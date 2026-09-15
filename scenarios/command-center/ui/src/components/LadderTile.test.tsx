import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { authoredSample, makeReading } from "../test-utils/readings";
import { ladderReading } from "../test-utils/ladder";
import { LadderTile } from "./LadderTile";
import { PanelTile } from "./PanelTile";

describe("LadderTile", () => {
  it("summarises the schedule by its next release instead of a dash", () => { // [REQ:CC-P1-016]
    const { container } = renderWithProviders(<ul><LadderTile reading={ladderReading()} /></ul>);
    expect(screen.getByText("web-console")).toBeInTheDocument();
    expect(screen.getByText(/Trigger met · next of 5/)).toBeInTheDocument();
    expect(screen.getByText("4 of 5 still ideas · 1 unscheduled")).toBeInTheDocument();
    expect(container.querySelectorAll(".cc-ladder-tile__seg")).toHaveLength(5);
    expect(container.querySelector(".cc-ladder-tile__seg[data-next]")).not.toBeNull();
  });
});

describe("PanelTile", () => {
  it("shows a panel's leading rows and how many more it has", () => { // [REQ:CC-P1-016]
    const rows = ["US", "DE", "FR", "JP"].map((key, index) => ({ key, label: key, value: 1000 - index * 100, share: 0.25 }));
    renderWithProviders(<ul><PanelTile reading={makeReading({ kind: "panel", label: "Traffic by country", rows })} /></ul>);
    expect(screen.getByText("US")).toBeInTheDocument();
    expect(screen.getByText("FR")).toBeInTheDocument();
    expect(screen.queryByText("JP")).toBeNull();
    expect(screen.getByText("+1 more")).toBeInTheDocument();
  });
  it("marks authored rows as a sample", () => {
    const { container } = renderWithProviders(<ul><PanelTile reading={makeReading({ kind: "panel", coverage: "IN-REACH", trust: "UNAVAILABLE", sample: { ...authoredSample(2), rows: [{ key: "a", label: "Control", value: 52, share: 0.52 }] } })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-provenance", "sample");
    expect(screen.getByText("Control")).toBeInTheDocument();
  });
  it("says it has no rows instead of drawing an empty list", () => {
    const { container } = renderWithProviders(<ul><PanelTile reading={makeReading({ kind: "panel", label: "Traffic by device" })} /></ul>);
    expect(screen.getByText("No rows available")).toBeInTheDocument();
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-provenance", "absent");
    expect(screen.queryByText(/more$/)).toBeNull();
  });
  it("marks measured rows as measured", () => {
    const { container } = renderWithProviders(<ul><PanelTile reading={makeReading({ kind: "panel", rows: [{ key: "m", label: "Mobile", value: 12, share: 1 }] })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-provenance", "measured");
  });
});
