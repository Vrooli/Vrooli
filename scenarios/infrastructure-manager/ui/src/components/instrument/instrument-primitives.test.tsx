import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders as render } from "../../test-utils";
import { strings } from "../../consts/strings.generated";
import { AnnunciatorGrid, type AnnunciatorRow } from "./AnnunciatorGrid";
import { Lamp } from "./Lamp";
import { LegendPlate } from "./LegendPlate";
import { RungRing, describeRungStates } from "./RungRing";
import { StatPlate, StatStrip } from "./StatPlate";
import {
  RUNG_ORDER,
  SIGNAL_STATES,
  enforceRungDependency,
  type Rung,
  type SignalState,
} from "../../theme/instrument";

const ring = (...states: SignalState[]): Record<Rung, SignalState> =>
  RUNG_ORDER.reduce(
    (acc, rung, index) => {
      acc[rung] = states[index] ?? "BLIND";
      return acc;
    },
    {} as Record<Rung, SignalState>,
  );

/**
 * These tests assert the design language's INVARIANTS, not its markup. Each
 * one corresponds to a claim the scenario makes about honesty: status is never
 * colour alone, absence is never rendered as zero, an instrument fault is
 * never rendered as a plant fault, and a rung never claims coverage over a
 * blind foundation.
 */
describe("annunciator lamp", () => {
  it("names its state in words, so status is never carried by colour alone", () => {
    render(<Lamp state="BLIND" subject="storage, Anticipation" />);
    const lamp = screen.getByRole("img");
    expect(lamp).toHaveAccessibleName(/storage, Anticipation/);
    expect(lamp).toHaveAccessibleName(/Blind/);
  });

  it("carries a distinct mark for every state, so states differ without colour", () => {
    const marks = Object.values(SIGNAL_STATES).map((token) => token.mark);
    expect(new Set(marks).size).toBe(marks.length);
    const shorts = Object.values(SIGNAL_STATES).map((token) => token.short);
    expect(new Set(shorts).size).toBe(shorts.length);
  });

  it("states how long a blind lamp has been blind", () => {
    // Tests run in i18next `cimode`, where `t()` returns the key rather than
    // the interpolated copy — deliberately, so tests survive copy changes.
    // Asserting the key proves the phrase is wired; asserting that the two
    // renders differ proves the day count actually reaches the caller.
    const { rerender } = render(
      <Lamp state="BLIND" subject="integrated graphics, Identity" blindDays={114} />,
    );
    const withAge = screen.getByRole("img").getAttribute("aria-label");
    expect(withAge).toContain(strings.instrument.blindFor);

    rerender(<Lamp state="BLIND" subject="integrated graphics, Identity" />);
    expect(screen.getByRole("img").getAttribute("aria-label")).not.toEqual(withAge);
  });

  it("surfaces the reason on an unmeasurable lamp, which is what makes it honest", () => {
    render(
      <Lamp
        state="UNMEASURABLE"
        subject="nvme0n1, Anticipation"
        reason="smartctl present, permission denied"
      />,
    );
    expect(screen.getByRole("img")).toHaveAccessibleName(/permission denied/);
  });

  it("renders UNAVAILABLE distinctly from BLIND, so an instrument outage cannot read as a coverage collapse", () => {
    const { rerender } = render(<Lamp state="UNAVAILABLE" subject="device graph source" />);
    const unavailable = screen.getByRole("img").className;
    rerender(<Lamp state="BLIND" subject="device graph source" />);
    const blind = screen.getByRole("img").className;
    expect(unavailable).not.toEqual(blind);
  });

  it("renders UNMEASURABLE distinctly from both BLIND and COVERED", () => {
    const classNames = new Set<string>();
    for (const state of ["UNMEASURABLE", "BLIND", "COVERED"] as const) {
      const { unmount } = render(<Lamp state={state} subject="s" />);
      classNames.add(screen.getByRole("img").className);
      unmount();
    }
    expect(classNames.size).toBe(3);
  });
});

describe("legend plate", () => {
  it("renders a tag only when the section has a real reference to carry", () => {
    const { container, rerender } = render(<LegendPlate tag="SB1" legend="Substrate" />);
    expect(container.querySelector(".plate__tag")).toHaveTextContent("SB1");
    rerender(<LegendPlate legend="Substrate" />);
    expect(container.querySelector(".plate__tag")).toBeNull();
  });

  it("lets a caller step the heading level so the document outline stays correct", () => {
    render(<LegendPlate as="h3" legend="Anticipation" />);
    expect(screen.getByRole("heading", { level: 3, name: "Anticipation" })).toBeInTheDocument();
  });
});

describe("stat plate", () => {
  it("renders an em dash for a figure that could not be computed, never a zero", () => {
    const { container } = render(
      <StatStrip label="figures">
        <StatPlate value={null} label="Rungs covered" />
      </StatStrip>,
    );
    const value = container.querySelector(".stat__value");
    expect(value).toHaveTextContent("—");
    expect(value).not.toHaveTextContent("0");
  });
});

describe("rung ring", () => {
  it("draws one segment per ladder rung", () => {
    const { container } = render(
      <svg>
        <RungRing cx={50} cy={50} radius={36} states={ring("COVERED", "COVERED", "BLIND", "BLIND", "BLIND")} />
      </svg>,
    );
    expect(container.querySelectorAll("circle")).toHaveLength(RUNG_ORDER.length);
  });

  it("describes the same finding in words that the shape shows", () => {
    const description = describeRungStates(
      "AMD Raphael",
      ring("BLIND", "BLIND", "BLIND", "BLIND", "BLIND"),
    );
    expect(description).toContain("0 of 5 rungs covered");
    expect(description).toContain("Identity blind");
  });
});

describe("rung dependency rule", () => {
  it("demotes a rung that claims coverage over a blind foundation", () => {
    const graded = enforceRungDependency(ring("BLIND", "COVERED", "COVERED", "COVERED", "COVERED"));
    expect(graded.IDENTITY).toBe("BLIND");
    expect(graded.TELEMETRY).toBe("PARTIAL");
    expect(graded.ANTICIPATION).toBe("PARTIAL");
  });

  it("leaves a fully covered ladder untouched", () => {
    const states = ring("COVERED", "COVERED", "COVERED", "COVERED", "COVERED");
    expect(enforceRungDependency(states)).toEqual(states);
  });

  it("does not demote states that already state their own limit", () => {
    const graded = enforceRungDependency(
      ring("BLIND", "UNMEASURABLE", "EXCURSION", "UNAVAILABLE", "PARTIAL"),
    );
    expect(graded.TELEMETRY).toBe("UNMEASURABLE");
    expect(graded.EVIDENCE).toBe("EXCURSION");
    expect(graded.CONTROL).toBe("UNAVAILABLE");
  });
});

describe("annunciator grid", () => {
  const rows: readonly AnnunciatorRow[] = [
    {
      id: "nvme",
      name: "Samsung SSD 990 PRO",
      tag: "pci:0000:02:00.0",
      states: ring("COVERED", "COVERED", "PARTIAL", "COVERED", "UNMEASURABLE"),
      reasons: { ANTICIPATION: "smartctl present, permission denied" },
    },
  ];

  it("is a real table with both header axes, so a screen reader can announce the finding", () => {
    render(<AnnunciatorGrid rows={rows} rowHeader="Device" caption="Fixture data." />);
    const table = screen.getByRole("table");
    expect(within(table).getAllByRole("columnheader")).toHaveLength(RUNG_ORDER.length + 1);
    expect(within(table).getByRole("rowheader")).toHaveTextContent("Samsung SSD 990 PRO");
  });

  it("carries every rung's reason through to the lamp", () => {
    render(<AnnunciatorGrid rows={rows} rowHeader="Device" caption="Fixture data." />);
    expect(
      screen.getByRole("img", { name: /Anticipation, Unmeasurable, smartctl present, permission denied/ }),
    ).toBeInTheDocument();
  });

  it("states its denominator in the caption rather than printing a bare matrix", () => {
    const { container } = render(
      <AnnunciatorGrid rows={rows} rowHeader="Device" caption="Fixture data." />,
    );
    expect(container.querySelector("caption")).toHaveTextContent("Fixture data.");
  });
});
