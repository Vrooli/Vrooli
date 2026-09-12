import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DeviceConstellation, describeConstellation } from "./DeviceConstellation";
import { DeviceDrilldown, type DeviceDrilldownLabels } from "./DeviceDrilldown";
import { PortabilityLegend, PortabilityMatrix, QUALIFICATION_ORDER } from "./PortabilityMatrix";
import { RUNG_ORDER, type Rung, type SignalState } from "../../theme/instrument";
import type { DeviceClassNode, RungDetail } from "./model";
import { renderWithProviders as render } from "../../test-utils";

/**
 * Branch coverage for the board's presentational pieces.
 *
 * Each case here is a BRANCH THAT MATTERS: a value that is absent versus
 * present, a class that is seen versus unseen, a qualification rung nobody
 * handled. Every one of them is a place where a component could quietly render
 * something more flattering than the truth.
 */

function detail(state: SignalState, overrides: Partial<RungDetail> = {}): RungDetail {
  return {
    state,
    cellRef: null,
    question: null,
    reason: null,
    mechanism: null,
    remediation: null,
    blockedBy: null,
    trust: null,
    graded: true,
    ungradedReason: null,
    provisional: false,
    blindDays: null,
    ...overrides,
  };
}

function node(
  deviceClass: string,
  states: SignalState[],
  overrides: Partial<DeviceClassNode> = {},
): DeviceClassNode {
  const rungs = RUNG_ORDER.reduce(
    (acc, rung, index) => {
      acc[rung] = detail(states[index] ?? "BLIND");
      return acc;
    },
    {} as Record<Rung, RungDetail>,
  );
  return { deviceClass, rungs, deviceCount: 2, blindDevices: 0, ...overrides };
}

const COVERED: SignalState[] = ["COVERED", "COVERED", "COVERED", "COVERED", "COVERED"];
const BLIND: SignalState[] = ["BLIND", "BLIND", "BLIND", "BLIND", "BLIND"];

const LABELS: DeviceDrilldownLabels = {
  heading: "Identity and ladder",
  ladder: "Observability ladder",
  devices: "Devices enumerated",
  blindDevices: "With no covered rung",
  reasonHeading: "WHY",
  remediationHeading: "TO CLOSE",
  mechanismHeading: "READ FROM",
  noRemediation: "No remediation is declared yet.",
  ungraded: "Not graded against a setpoint bar",
  provisional: "The bar behind this cell is still provisional.",
  blockedBy: (rung) => `Suppressed because ${rung} below it is blind.`,
  blindFor: (days) => `Longest-standing blindness: ${days} days.`,
  notRead: "The source did not read this.",
};

describe("device drilldown", () => {
  it("renders an em dash for a device count the source did not read", () => {
    const { container } = render(
      <DeviceDrilldown node={node("storage", COVERED, { deviceCount: null, blindDevices: null })} labels={LABELS} />,
    );
    expect(container.querySelectorAll("dd")[0]).toHaveTextContent("—");
  });

  it("states the remediation when one is declared, and says so when none is", () => {
    const withRemediation = node("storage", COVERED);
    const rungs = { ...withRemediation.rungs };
    const MECHANISM = "smartctl -j -A";
    rungs.ANTICIPATION = detail("UNMEASURABLE", {
      reason: "permission denied",
      remediation: "commission the smartctl host tool",
      mechanism: MECHANISM,
    });
    rungs.CONTROL = detail("BLIND");
    render(<DeviceDrilldown node={{ ...withRemediation, rungs }} labels={LABELS} />);
    expect(screen.getByText(/commission the smartctl host tool/)).toBeInTheDocument();
    expect(screen.getByText(LABELS.noRemediation)).toBeInTheDocument();
    expect(screen.getByText(MECHANISM)).toBeInTheDocument();
  });

  it("names the lower rung that suppressed a blocked grade", () => {
    const blocked = node("storage", COVERED);
    const rungs = { ...blocked.rungs };
    rungs.ANTICIPATION = detail("PARTIAL", { blockedBy: "IDENTITY" });
    render(<DeviceDrilldown node={{ ...blocked, rungs }} labels={LABELS} />);
    expect(screen.getByText(/Suppressed because Identity below it is blind/)).toBeInTheDocument();
  });

  it("says a measured reading is ungraded when no bar resolves", () => {
    // Measured-but-ungraded is not passing, and the panel must not let it read
    // as a verdict.
    const ungraded = node("thermal", COVERED);
    const rungs = { ...ungraded.rungs };
    rungs.TELEMETRY = detail("COVERED", { graded: false, ungradedReason: "no bar resolves" });
    render(<DeviceDrilldown node={{ ...ungraded, rungs }} labels={LABELS} />);
    expect(screen.getByText(/Not graded against a setpoint bar — no bar resolves/)).toBeInTheDocument();
  });

  it("flags a provisional bar", () => {
    const provisional = node("storage", COVERED);
    const rungs = { ...provisional.rungs };
    rungs.IDENTITY = detail("COVERED", { provisional: true });
    render(<DeviceDrilldown node={{ ...provisional, rungs }} labels={LABELS} />);
    expect(screen.getByText(LABELS.provisional)).toBeInTheDocument();
  });

  it("states the longest-standing blindness when any rung is dated", () => {
    const dated = node("memory", BLIND);
    const rungs = { ...dated.rungs };
    rungs.ANTICIPATION = detail("BLIND", { blindDays: 114 });
    render(<DeviceDrilldown node={{ ...dated, rungs }} labels={LABELS} />);
    expect(screen.getByText(/Longest-standing blindness: 114 days/)).toBeInTheDocument();
  });

  it("shows the cell reference when the rung answers an authored cell", () => {
    const referenced = node("storage", COVERED);
    const rungs = { ...referenced.rungs };
    const QUESTION = "Is it identified?";
    rungs.IDENTITY = detail("COVERED", { cellRef: "substrate/SB9", question: QUESTION });
    render(<DeviceDrilldown node={{ ...referenced, rungs }} labels={LABELS} />);
    expect(screen.getByText(/substrate\/SB9/)).toBeInTheDocument();
    expect(screen.getByText(QUESTION)).toBeInTheDocument();
  });
});

describe("device constellation", () => {
  const classes = [node("block-device", COVERED), node("thermal-sensor", BLIND, { deviceCount: 8 })];
  const summary = describeConstellation("linux", classes);

  it("is operable by keyboard and reports its pressed state", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    render(
      <DeviceConstellation
        hostName="linux"
        classes={classes}
        onSelectClass={onSelect}
        selectedClass="block-device"
        summary={summary}
      />,
    );
    const nodes = screen.getAllByRole("button");
    expect(nodes[0]).toHaveAttribute("aria-pressed", "true");
    nodes[1]!.focus();
    await user.keyboard("{Enter}");
    expect(onSelect).toHaveBeenCalledWith("thermal-sensor");
    await user.keyboard(" ");
    expect(onSelect).toHaveBeenCalledTimes(2);
  });

  it("renders no interactive node when no handler is supplied", () => {
    render(<DeviceConstellation hostName="linux" classes={classes} summary={summary} />);
    expect(screen.queryAllByRole("button")).toHaveLength(0);
  });

  it("carries a title and a description so the shape has a text alternative", () => {
    render(<DeviceConstellation hostName="linux" classes={classes} summary={summary} />);
    const figure = screen.getByRole("img");
    expect(figure).toHaveAccessibleName(/linux/);
  });

  it("collapses to a short band when there is nothing to place", () => {
    const { container } = render(
      <DeviceConstellation hostName="linux" classes={[]} summary={describeConstellation("linux", [])} />,
    );
    // A full-height empty field reads as a broken chart; a short one reads as
    // the instrument stating it read nothing.
    expect(container.querySelector("svg")).toHaveAttribute("viewBox", "0 0 920 220");
  });
});

describe("portability matrix", () => {
  const rows = [
    {
      capability: "system-monitor-cpu",
      platforms: {
        linux: {
          status: "implemented",
          qualification: "qualified",
          implementer: "system-monitor",
          mechanism: "procfs",
          reason: "runs on real hardware",
        },
        macos: {
          status: "status_invalid",
          qualification: "not-a-real-rung",
          implementer: null,
          mechanism: null,
          reason: "the manifest authored an unrecognised status",
        },
      },
    },
  ];

  it("renders nothing-declared for an operating system with no entry", () => {
    render(
      <PortabilityMatrix
        rows={rows}
        operatingSystems={["linux", "macos", "windows"]}
        rowHeader="Capability"
        caption="fixture"
      />,
    );
    expect(
      screen.getByRole("img", { name: /pages.substrate.nothingDeclaredLabel/ }),
    ).toBeInTheDocument();
  });

  it("renders an unrecognised qualification as an excursion so it is visible", () => {
    // Never as the most flattering rung on the list.
    render(
      <PortabilityMatrix
        rows={rows}
        operatingSystems={["linux", "macos"]}
        rowHeader="Capability"
        caption="fixture"
      />,
    );
    const unknown = screen.getByRole("img", { name: /Unrecognised/ });
    expect(unknown.className).toMatch(/excursion/);
  });

  it("renders incomplete controls as a distinct state with absent names", () => {
    render(
      <PortabilityMatrix
        rows={[{
          capability: "credential-storage",
          platforms: { windows: {
            status: "controls_incomplete",
            qualification: "qualified",
            implementer: "credential-provider",
            mechanism: null,
            reason: "provider resolves but a control is absent",
            absent: ["login_keyring_unlock"],
          } },
        }]}
        operatingSystems={["windows"]}
        rowHeader="Capability"
        caption="fixture"
      />,
    );
    expect(screen.getByRole("img", { name: /Controls incomplete/ })).toHaveClass("lamp--excursion");
    expect(screen.getByTestId("absent-declarers")).toHaveTextContent("login_keyring_unlock");
  });

  it("states the denominator in its caption", () => {
    const { container } = render(
      <PortabilityMatrix
        rows={rows}
        operatingSystems={["linux"]}
        rowHeader="Capability"
        caption="41 capabilities resolved"
      />,
    );
    expect(container.querySelector("caption")).toHaveTextContent("41 capabilities resolved");
  });

  it("renders a key for every qualification rung", () => {
    const { container } = render(<PortabilityLegend rungs={QUALIFICATION_ORDER} />);
    expect(container.querySelectorAll("li")).toHaveLength(QUALIFICATION_ORDER.length);
  });

  it("renders a key entry for an unrecognised rung too", () => {
    const { container } = render(<PortabilityLegend rungs={["not-a-real-rung"]} />);
    // Queried by structure rather than by copy: the point is that an
    // unrecognised rung still gets a key entry, not what that entry says.
    expect(container.querySelectorAll("li")).toHaveLength(1);
    expect(container.querySelector(".lamp")?.className).toMatch(/excursion/);
  });
});
