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

  it("keeps the retained ladder explicitly unavailable with its last-good age", () => {
    const observedAt = new Date(Date.now() - 90 * 60_000).toISOString();
    const { container } = renderWithProviders(<ul><LadderTile reading={ladderReading({
      trust: "UNAVAILABLE", observedAt, value: 5, trustReason: "producer observation is stale",
    })} /></ul>);
    expect(screen.getByText("UNAVAILABLE", { selector: "strong" })).toBeInTheDocument();
    const trustLine = screen.getByRole("group", { name: /Release ladder: UNAVAILABLE.*producer observation is stale · last good 2h ago/ });
    expect(trustLine.querySelector("[data-qualifier]")).toHaveTextContent("source not answering · last good 2h ago");
    expect(trustLine.querySelector("[data-qualifier]")).toHaveAttribute("title", expect.stringContaining("producer observation is stale"));
    expect(screen.getByText("web-console")).toBeInTheDocument();
    expect(container.querySelector("[data-reading-trust=UNAVAILABLE]")).not.toBeNull();
  });
});

describe("PanelTile", () => {
  it("shows a panel's leading rows and how many more it has", () => { // [REQ:CC-P1-016]
    const rows = ["US", "DE", "FR", "JP"].map((key, index) => ({ key, label: key, value: 1000 - index * 100, share: 0.25 }));
    renderWithProviders(<ul><PanelTile reading={makeReading({ kind: "panel", label: "Traffic by country", value: 2700, rows })} /></ul>);
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
    expect(container.querySelector(".cc-panel-tile__empty")).toHaveTextContent(/not answering/);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-provenance", "absent");
    expect(screen.queryByText(/more$/)).toBeNull();
  });
  it("marks measured rows as measured", () => {
    const { container } = renderWithProviders(<ul><PanelTile reading={makeReading({ kind: "panel", value: 12, rows: [{ key: "m", label: "Mobile", value: 12, share: 1 }] })} /></ul>);
    expect(container.querySelector("[data-reading]")).toHaveAttribute("data-provenance", "measured");
  });

  it("withholds current panel rows until an untrusted source can be verified", () => {
    const { container } = renderWithProviders(<ul><PanelTile reading={makeReading({
      kind: "panel", label: "Readiness", coverage: "NOW", trust: "UNTRUSTED", value: 0,
      trustReason: "producer did not supply observation time", rows: [{ key: "missing", label: "Readiness goal missing", value: 0, share: 1 }],
    })} /></ul>);
    expect(screen.queryByText("Readiness goal missing")).toBeNull();
    expect(container.querySelector(".cc-panel-tile__rows")).toBeNull();
    expect(container.querySelector(".cc-panel-tile__empty")).toHaveTextContent("Not shown until the source reading can be verified.");
    expect(screen.getByRole("group", { name: /Readiness: UNTRUSTED.*producer did not supply observation time/ })).toBeInTheDocument();
    expect(container.querySelector("[data-qualifier]")).toHaveTextContent("no source time");
  });
});
