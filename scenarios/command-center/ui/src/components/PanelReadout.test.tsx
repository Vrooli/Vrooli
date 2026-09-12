import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { makeReading } from "../test-utils/readings";
import { PanelReadout } from "./PanelReadout";

describe("PanelReadout", () => {
  it("renders authored sample rows when measured rows are unavailable", () => {
    renderWithProviders(<PanelReadout reading={makeReading({ kind: "panel", sample: { value: 840, series: [840], basis: "reviewed", rows: [{ key: "US", label: "United States", value: 420, share: 0.5 }] } })} />);
    expect(screen.getByText("United States")).toBeInTheDocument();
    expect(screen.queryByLabelText("No observations")).toBeNull();
  });

  it("uses a plain text empty state instead of a bordered placeholder", () => {
    const { container } = renderWithProviders(<PanelReadout reading={makeReading({ kind: "panel" })} />);
    expect(screen.getByText("No rows available")).toBeInTheDocument();
    expect(container.querySelector(".cc-panel-readout__placeholder")).toBeNull();
  });
});
