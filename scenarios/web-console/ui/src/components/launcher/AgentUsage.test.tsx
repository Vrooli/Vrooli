import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import { CardUsage, UsageDetail } from "./AgentUsage";
import type { AgentUsage as Reading, AgentUsageWindow } from "./useAgentUsage";
import { reservedUsageRows, toneFor, usageRows, windowNames } from "./usageWindows";

const inFuture = (ms: number) => new Date(Date.now() + ms).toISOString();
const DAY = 86_400_000;
const windowOf = (over: Partial<AgentUsageWindow>): AgentUsageWindow => ({ pool: "primary", windowMinutes: 10080, usedPercent: 42, resetsAt: inFuture(DAY), observedAt: new Date().toISOString(), ...over });
const reading = (windows: AgentUsageWindow[], over: Partial<Reading> = {}): Reading => ({
  targetId: "local", agentId: "codex", match: "likely", reason: "", source: "provider", observedAt: new Date().toISOString(), windows, ...over,
});
const rows = (kind: string) => document.querySelectorAll(`[data-usage-row="${kind}"]`);

describe("card usage", () => {
  it("draws a row for each window the provider reports, including durations nobody hardcoded", () => {
    render(<CardUsage usage={reading([windowOf({ windowMinutes: 10080 }), windowOf({ windowMinutes: 300 }), windowOf({ windowMinutes: 90 })])} loading={false} slots={3} declared={2} />);
    const meters = screen.getAllByRole("progressbar");
    expect(meters).toHaveLength(3);
    expect(meters.map((meter) => meter.getAttribute("aria-valuenow"))).toEqual(["42", "42", "42"]);
    // Shortest first, whichever order they arrive in.
    expect(meters[0]).toHaveAccessibleName(/launcher\.usageRow/);
    expect(meters[0]?.closest('[data-usage-row="reading"]')).toHaveTextContent(windowNames(90, { week: "w", weekly: "ww" }, "en").short);
  });

  it("does not invent a window the provider stopped reporting", () => {
    render(<CardUsage usage={reading([windowOf({ windowMinutes: 10080 })])} loading={false} slots={2} declared={2} />);
    expect(screen.getAllByRole("progressbar")).toHaveLength(1);
    expect(rows("blank")).toHaveLength(1);
  });

  // The reservation is the whole point: a state never changes how tall a card is.
  it.each([
    ["loaded", { usage: reading([windowOf({ windowMinutes: 300 }), windowOf({})]), loading: false, declared: 2 }],
    ["one window", { usage: reading([windowOf({})]), loading: false, declared: 2 }],
    ["loading", { usage: undefined, loading: true, declared: 2 }],
    ["loading with fewer declared", { usage: undefined, loading: true, declared: 1 }],
    ["unavailable", { usage: reading([], { match: "unavailable" }), loading: false, declared: 2 }],
    ["failed", { usage: null, loading: false, declared: 2 }],
    ["no usage declared", { usage: null, loading: false, declared: 0 }],
    ["stale", { usage: reading([windowOf({ observedAt: inFuture(-7 * 3_600_000) })]), loading: false, declared: 1 }],
  ] as const)("occupies every reserved row while %s", (_name, props) => {
    render(<CardUsage {...props} slots={2} />);
    expect(screen.getByTestId("launcher-card-usage").children).toHaveLength(2);
  });

  it("keeps a reading on screen during a refresh instead of a placeholder", () => {
    render(<CardUsage usage={reading([windowOf({ usedPercent: 87 })])} loading slots={1} declared={1} />);
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "87");
    expect(rows("placeholder")).toHaveLength(0);
  });

  it("does not present an old reading or a passed reset as current", () => {
    const old = reading([windowOf({ observedAt: inFuture(-7 * 3_600_000) })]);
    const { rerender } = render(<CardUsage usage={old} loading={false} slots={1} declared={1} />);
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.getByText("launcher.usageStale")).toBeInTheDocument();
    rerender(<CardUsage usage={reading([windowOf({ resetsAt: inFuture(-1000) })])} loading={false} slots={1} declared={1} />);
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });

  it("ignores windows that are not readable percentages", () => {
    expect(usageRows(reading([windowOf({ usedPercent: 140 }), windowOf({ usedPercent: Number.NaN }), windowOf({ windowMinutes: 0 })]))).toEqual([]);
  });

  it("shows the tightest pool for a duration and says others exist, never a sum", () => {
    render(<CardUsage usage={reading([windowOf({ pool: "a", usedPercent: 8 }), windowOf({ pool: "b", usedPercent: 60 })])} loading={false} slots={1} declared={1} />);
    const meter = screen.getByRole("progressbar");
    expect(meter).toHaveAttribute("aria-valuenow", "60");
    expect(meter.closest('[data-usage-row="reading"]')).toHaveTextContent("launcher.usageOtherPools");
  });

  it("colours a bar amber at 80% and red at 95%", () => {
    expect([79.9, 80, 94.9, 95, 100].map(toneFor)).toEqual(["normal", "hot", "hot", "full", "full"]);
    render(<CardUsage usage={reading([windowOf({ windowMinutes: 300, usedPercent: 96 }), windowOf({ usedPercent: 85 })])} loading={false} slots={2} declared={2} />);
    expect(screen.getAllByRole("progressbar").map((meter) => meter.getAttribute("data-usage-tone"))).toEqual(["full", "hot"]);
  });
});

describe("window names", () => {
  const words = { week: "Week", weekly: "Weekly" };
  it("names windows from their duration", () => {
    expect(windowNames(300, words, "en")).toEqual({ short: "5h", long: "5 hours" });
    expect(windowNames(10080, words, "en")).toEqual({ short: "Week", long: "Weekly" });
    expect(windowNames(1440, words, "en").long).toBe("1 day");
    expect(windowNames(45, words, "en").short).toBe("45m");
  });
});

describe("reserved rows", () => {
  it("comes from the largest declared count among visible harnesses, before any reading", () => {
    expect(reservedUsageRows([2, 1, 0], [undefined, undefined, undefined])).toBe(2);
    expect(reservedUsageRows([0, 0], [null, null])).toBe(1);
  });
  it("grows for a provider that reports more than it declared, so nothing is hidden", () => {
    expect(reservedUsageRows([1], [reading([windowOf({ windowMinutes: 300 }), windowOf({})])])).toBe(2);
  });
});

describe("usage in Manage", () => {
  it("lists each pool separately with absolute reset times and one source sentence", () => {
    render(<UsageDetail usage={reading([windowOf({ pool: "b", usedPercent: 8 }), windowOf({ pool: "a", usedPercent: 42 })])} loading={false} declared={1} agentLabel="Codex" />);
    const detail = screen.getByTestId("launcher-usage-codex");
    expect(within(detail).getAllByRole("progressbar").map((bar) => bar.getAttribute("aria-valuenow"))).toEqual(["42", "8"]);
    expect(within(detail).getAllByText("launcher.usageResets")).toHaveLength(2);
    expect(screen.getByTestId("launcher-usage-source")).toHaveTextContent("launcher.usageSourceLikely");
  });

  it("states that a harness reports nothing, and passes on a server's reason when it does", () => {
    const { rerender } = render(<UsageDetail usage={null} loading={false} declared={0} agentLabel="Grok" />);
    expect(screen.getByText("launcher.usageNotReportedFor")).toBeInTheDocument();
    rerender(<UsageDetail usage={reading([], { match: "unavailable", reason: "Quota feed is down" })} loading={false} declared={2} agentLabel="Claude" />);
    expect(screen.getByText("Quota feed is down")).toBeInTheDocument();
  });
});
