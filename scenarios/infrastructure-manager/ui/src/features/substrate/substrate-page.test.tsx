import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings.generated";
import { RUNG_ORDER, type Rung, type SignalState } from "../../theme/instrument";
import type { DeviceClassNode, RungDetail, SubstrateBoard } from "./model";

const mocks = vi.hoisted(() => ({ fetchSubstrateBoard: vi.fn() }));
vi.mock("../../api/substrate", () => mocks);

import { SubstrateBoardPage } from "./SubstrateBoardPage";

/**
 * Render tests for the Substrate Board.
 *
 * These assert the board's HONESTY CONTRACT at the rendered surface, not its
 * markup: that an unread source produces an em dash rather than a zero, that a
 * blind class is visibly distinct from an undeclared one, and that no control
 * anywhere mutates anything. Copy is asserted through `strings.*` because tests
 * run in i18next `cimode`, where `t()` returns the key.
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

function board(overrides: Partial<SubstrateBoard> = {}): SubstrateBoard {
  return {
    host: { name: "linux", os: "linux" },
    classes: [
      node("block-device", ["COVERED", "COVERED", "COVERED", "COVERED", "UNMEASURABLE"], {
        rungs: {
          ...node("block-device", COVERED).rungs,
          ANTICIPATION: detail("UNMEASURABLE", {
            reason: "permission denied",
            remediation: "commission the smartctl host tool",
            cellRef: "substrate/SB10",
          }),
        },
      }),
      node("thermal-sensor", ["BLIND", "BLIND", "BLIND", "BLIND", "BLIND"], {
        deviceCount: 8,
        blindDevices: 8,
      }),
    ],
    portability: [
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
            status: "implemented",
            qualification: "build-verified",
            implementer: "system-monitor",
            mechanism: "host_statistics",
            reason: "compiles and passes fixtures; never run on a real host",
          },
        },
      },
    ],
    sources: [
      { name: "device-graph", verdict: "VALID", reason: null },
      { name: "portability", verdict: "VALID", reason: "41 manifests read" },
    ],
    checkPlatforms: [
      { hostOs: "linux", applicable: 19, total: 24, universal: 5, available: true, reason: null },
    ],
    coverageAvailable: true,
    coverageReason: null,
    denominator: { confidence: "PARTIAL", rationale: "eight of thirteen cells have a sensor" },
    ...overrides,
  };
}

function renderBoard() {
  return renderWithProviders(<SubstrateBoardPage />, {
    queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  });
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("substrate board", () => {
  it("renders every class with its rung lamps", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    renderBoard();
    const matrix = await screen.findByTestId("substrate-rung-matrix");
    // Awaited INSIDE the scope: the surface element renders during the loading
    // state, so resolving the testid alone does not mean the data has arrived.
    const grid = await within(matrix).findByRole("table");
    expect(within(grid).getAllByRole("rowheader")).toHaveLength(2);
    // One column per rung, plus the device column.
    expect(within(grid).getAllByRole("columnheader")).toHaveLength(RUNG_ORDER.length + 1);
  });

  it("carries an unmeasurable rung's reason through to its lamp", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    renderBoard();
    expect(
      await screen.findByRole("img", { name: /Anticipation, Unmeasurable, permission denied/ }),
    ).toBeInTheDocument();
  });

  it("names each source and its verdict on the instrument chrome", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    renderBoard();
    expect(await screen.findByText(/device-graph · VALID/)).toBeInTheDocument();
  });

  it("states plainly when the space document itself could not be read", async () => {
    const REASON = "the space verb did not answer";
    mocks.fetchSubstrateBoard.mockResolvedValue(
      board({ coverageAvailable: false, coverageReason: REASON }),
    );
    renderBoard();
    expect(await screen.findByText(REASON)).toBeInTheDocument();
  });

  it("renders an em dash rather than a zero when the ladder read nothing", async () => {
    // The single most important assertion on this page: "no classes" and "we
    // could not look" must never render the same way.
    mocks.fetchSubstrateBoard.mockResolvedValue(board({ classes: [] }));
    renderBoard();
    const strip = await screen.findByLabelText(strings.pages.substrate.headlineLabel);
    expect(within(strip).getAllByLabelText(strings.instrument.notAvailable).length).toBeGreaterThan(0);
    expect(within(strip).queryByText("0")).not.toBeInTheDocument();
  });

  it("says it read nothing rather than implying the machine is empty", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board({ classes: [] }));
    renderBoard();
    const region = await screen.findByTestId("substrate-constellation");
    // The sentence appears twice by design — once as the SVG's <desc> and once
    // as the visible text alternative beside it — so this matches on the node's
    // own text content rather than asserting a single element.
    // The sentence appears TWICE by design — once as the SVG's <desc>, which
    // assistive technology reads, and once as the visible text alternative
    // beside it. Both are required, so this asserts both are present.
    const statements = await within(region).findAllByText(
      /not the same as the machine having nothing attached/,
    );
    expect(statements).toHaveLength(2);
  });

  it("names the failure instead of fabricating a board when the read throws", async () => {
    mocks.fetchSubstrateBoard.mockRejectedValue(new Error("ladder unreachable"));
    renderBoard();
    expect((await screen.findAllByText(strings.pages.substrate.unavailableTitle)).length).toBeGreaterThan(0);
  });

  it("renders build-verified distinctly from qualified in the portability matrix", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    renderBoard();
    const qualified = await screen.findByRole("img", { name: /system-monitor-cpu on linux, Qualified/ });
    const built = await screen.findByRole("img", { name: /system-monitor-cpu on macos, Build verified/ });
    expect(qualified.className).not.toEqual(built.className);
  });

  it("states substrate sensing with its denominator, never a bare count", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    renderBoard();
    expect(await screen.findByText(strings.pages.substrate.checkPlatformRow)).toBeInTheDocument();
  });

  it("opens a drill-down that names the remediation for an uncovered rung", async () => {
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    const user = userEvent.setup();
    renderBoard();
    const matrix = await screen.findByTestId("substrate-rung-matrix");
    const rowButtons = await within(matrix).findAllByRole("button");
    await user.click(rowButtons[0]!);
    expect(await screen.findByText(/commission the smartctl host tool/)).toBeInTheDocument();
  });

  it("offers no control that mutates anything", async () => {
    // Operating-model rule 3: this scenario has no actuation right. Every
    // button on the page may only change what is displayed.
    mocks.fetchSubstrateBoard.mockResolvedValue(board());
    renderBoard();
    const matrix = await screen.findByTestId("substrate-rung-matrix");
    await within(matrix).findByRole("table");
    const labels = screen
      .getAllByRole("button")
      .map((element) => `${element.textContent} ${element.getAttribute("aria-label") ?? ""}`);
    for (const label of labels) {
      expect(label).not.toMatch(/restart|reconcile|delete|remove|shelve|dismiss|apply|save|edit/i);
    }
  });
});
