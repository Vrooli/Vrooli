import { describe, expect, it } from "vitest";
import type { Constellation } from "../lib/api";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { makeReading } from "../test-utils/readings";
import { SkyReadout } from "./SkyReadout";

const board: Constellation[] = [
  { room: { id: "hive", title: "The Hive" }, readings: [makeReading({ id: "a", value: 4 }), makeReading({ id: "b", coverage: "IN-REACH" })] },
  { room: { id: "ledger", title: "Ledger" }, readings: [makeReading({ id: "a", value: 4 }), makeReading({ id: "c", trust: "UNAVAILABLE" }), makeReading({ id: "d", coverage: "MISSING", trust: "UNAVAILABLE" })] },
];

describe("SkyReadout", () => {
  it("leads with the measured count and names the whole it is counted from", () => {
    renderWithProviders(<SkyReadout constellations={board} />);
    expect(screen.getByLabelText("1").closest("[data-figure]")).toHaveAttribute("data-ink", "solid");
    expect(screen.getByText("Signals measured")).toBeInTheDocument();
    expect(screen.getByText(/of 4 registered across 2 rooms/)).toBeInTheDocument();
  });
  it("gives every unmeasured signal a reason, each in its own material", () => {
    renderWithProviders(<SkyReadout constellations={board} />);
    const tally = screen.getByTestId("sky-tally");
    for (const [state, text, ink] of [["failing", "1sensor failing", "unavailable"], ["in-reach", "1in reach", "hollow"], ["missing", "1no substrate", "dotted"]] as const) {
      const row = tally.querySelector(`[data-state="${state}"]`);
      expect(row?.textContent).toBe(text);
      expect(row?.querySelector(`[data-ink="${ink}"]`)).not.toBeNull();
    }
  });
});
