import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { makeReading } from "../test-utils/readings";
import { PanelReadout } from "./PanelReadout";

describe("PanelReadout", () => {
  it("renders authored sample rows only for an in-reach reading", () => {
    renderWithProviders(<PanelReadout reading={makeReading({ kind: "panel", coverage: "IN-REACH", trust: "UNAVAILABLE", sample: { value: 840, series: [840], basis: "reviewed", rows: [{ key: "US", label: "United States", value: 420, share: 0.5 }] } })} />);
    expect(screen.getByText("United States")).toBeInTheDocument();
    expect(screen.queryByLabelText("No observations")).toBeNull();
  });

  it("uses a plain text empty state instead of a bordered placeholder", () => {
    const { container } = renderWithProviders(<PanelReadout reading={makeReading({ kind: "panel" })} />);
    expect(screen.getByText(/swarm-manager not answering · no value asserted/)).toBeInTheDocument();
    expect(container.querySelector(".cc-panel-readout__placeholder")).toBeNull();
  });

  it("explains empty Broadcast activity panels", () => {
    renderWithProviders(<PanelReadout reading={makeReading({ id: "credit_burn_by_app", label: "Credit burn by app", kind: "panel" })} />);
    expect(screen.getByText(/No credit burn recorded by app/)).toBeInTheDocument();
    expect(screen.getByText(/not answering/)).toBeInTheDocument();
  });

  it("labels retained rows as cached and carries the producer rejection reason", () => {
    const observedAt = new Date(Date.now() - 4 * 60_000).toISOString();
    const { container } = renderWithProviders(<PanelReadout reading={makeReading({
      kind: "panel", coverage: "NOW", trust: "CACHED", value: 100, observedAt, trustReason: "panel rows violated the share contract", sample: null,
      rows: [{ key: "us", label: "United States", value: 70, share: .7 }, { key: "de", label: "Germany", value: 30, share: .3 }],
    })} />);
    expect(screen.getByText("CACHED", { selector: "strong" })).toBeInTheDocument();
    expect(screen.getByText(/last good 4m ago · panel rows violated the share contract/)).toBeInTheDocument();
    expect(screen.getByText("United States")).toBeInTheDocument();
    expect(container.querySelector(".cc-panel-readout__row--reduced")).not.toBeNull();
  });

  it("states when retained panel data has aged out, including its last-good age", () => {
    const observedAt = new Date(Date.now() - 3 * 60_000).toISOString();
    const { container } = renderWithProviders(<PanelReadout reading={makeReading({
      kind: "panel", coverage: "NOW", trust: "UNAVAILABLE", value: 100, observedAt, trustReason: "producer observation is stale", sample: null,
      rows: [{ key: "us", label: "United States", value: 100, share: 1 }],
    })} />);
    expect(screen.getByText("UNAVAILABLE", { selector: "strong" })).toBeInTheDocument();
    const trustLine = screen.getByRole("group", { name: /Metric: UNAVAILABLE.*producer observation is stale · last good 3m ago/ });
    expect(trustLine.querySelector("[data-qualifier]")).toHaveTextContent("source not answering · last good 3m ago");
    expect(trustLine.querySelector("[data-qualifier]")).toHaveAttribute("title", expect.stringContaining("producer observation is stale"));
    expect(screen.queryByText("United States")).toBeNull();
    expect(container.querySelector("[data-trust=UNAVAILABLE]")).toHaveTextContent(/withheld after its observation expires/);
  });

  it("withholds rows whose current source reading is untrusted", () => {
    const { container } = renderWithProviders(<PanelReadout reading={makeReading({
      kind: "panel", label: "Readiness", coverage: "NOW", trust: "UNTRUSTED", value: 0,
      trustReason: "producer did not supply observation time", rows: [{ key: "missing", label: "Readiness goal missing", value: 0, share: 1 }],
    })} />);
    expect(screen.queryByText("Readiness goal missing")).toBeNull();
    expect(container.querySelector(".cc-panel-readout__rows")).toBeNull();
    expect(screen.getByText("Not shown until the source reading can be verified.")).toBeInTheDocument();
    const trustLine = screen.getByRole("group", { name: /Readiness: UNTRUSTED.*producer did not supply observation time/ });
    expect(trustLine.querySelector("[data-qualifier]")).toHaveTextContent("no source time");
  });

  it("shows supporting goal progress and empty-scope details", () => {
    renderWithProviders(<PanelReadout reading={makeReading({
      id: "goal_progress", label: "Goal progress", kind: "panel", coverage: "NOW", trust: "VALID", value: 2, sample: null,
      rows: [
        { key: "unscoped", label: "Unscoped goal", value: 0, share: 0, detail: "No scoped work" },
        { key: "delivery", label: "Delivery", value: 2, share: .4, detail: "4.0h–9.0h ETA" },
      ],
    })} />);

    expect(screen.getByText("No scoped work")).toBeInTheDocument();
    expect(screen.getByText("4.0h–9.0h ETA")).toBeInTheDocument();
  });
});
