import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { hueFor, initials, threadTitle, truncate } from "../../lib/identity";
import { msUntil, relativeTime, sameDay } from "../../lib/time";
import { AgentMark } from "./AgentMark";
import { budgetPressure } from "./BudgetMeter";
import { ChannelChip } from "./ChannelChip";
import { Quiet, Region } from "./Region";
import { TierBadge, tierRank } from "./TierBadge";

describe("console primitives", () => {
  it("renders an agent mark from the appearance triple with a live ring", () => {
    renderWithProviders(<AgentMark name="Household Planner" appearance={{ body: "#111", head: "#222", accent: "#333" }} live testId="mark" />);
    const mark = screen.getByTestId("mark");
    expect(mark).toHaveAttribute("aria-label", "Household Planner");
    expect(mark).toHaveAttribute("data-live", "true");
  });

  it("names the channel without relying on colour", () => {
    renderWithProviders(<ChannelChip id="telegram" name="Telegram" accent="#2AABEE" testId="chip" />);
    expect(screen.getByTestId("chip")).toHaveTextContent("Telegram");
    expect(screen.getByTestId("chip")).toHaveAttribute("data-channel", "telegram");
  });

  it("ranks tiers in order", () => {
    renderWithProviders(<TierBadge tier="trusted" testId="tier" />);
    expect(screen.getByTestId("tier")).toHaveAttribute("data-tier", "trusted");
    expect(tierRank("stranger")).toBeLessThan(tierRank("known"));
    expect(tierRank("trusted")).toBeLessThan(tierRank("owner"));
  });

  it("derives budget pressure from the worse of turns and spend", () => {
    const base = { thread_id: "t", channel_id: "c", thread_key: "k", agent_id: "a", window_started_at: "", exhausted: false };
    expect(budgetPressure({ ...base, turn_budget: 20, used: 2, spend_cap_cents: 0, spent_cents: 0 }).tone).toBe("success");
    expect(budgetPressure({ ...base, turn_budget: 20, used: 15, spend_cap_cents: 0, spent_cents: 0 }).tone).toBe("warning");
    expect(budgetPressure({ ...base, turn_budget: 20, used: 1, spend_cap_cents: 100, spent_cents: 100 }).tone).toBe("danger");
    expect(budgetPressure({ ...base, turn_budget: 0, used: 0, spend_cap_cents: 0, spent_cents: 0 }).tone).toBe("neutral");
  });

  it("exposes the declared region state and swaps bodies", () => {
    const { rerender } = renderWithProviders(
      <Region surfaceId="x" state="loading" testId="region">
        <p data-testid="ready-body">ready body</p>
      </Region>,
    );
    expect(screen.getByTestId("region")).toHaveAttribute("data-experience-state", "loading");
    expect(screen.queryByTestId("ready-body")).not.toBeInTheDocument();
    rerender(
      <Region surfaceId="x" state="error" testId="region" errorDetail="500" onRetry={() => undefined}>
        <p data-testid="ready-body">ready body</p>
      </Region>,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("500");
    expect(screen.getByTestId("region-retry")).toBeInTheDocument();
    rerender(
      <Region surfaceId="x" state="empty" testId="region" empty={<Quiet title="nothing" testId="quiet" />}>
        <p data-testid="ready-body">ready body</p>
      </Region>,
    );
    expect(screen.getByTestId("quiet")).toHaveTextContent("nothing");
  });
});

describe("identity and time helpers", () => {
  it("builds initials and stable hues", () => {
    expect(initials("Sam Jones")).toBe("SJ");
    expect(initials("@sam")).toBe("SA");
    expect(initials("")).toBe("?");
    expect(hueFor("a")).toBe(hueFor("a"));
    expect(hueFor("a")).not.toBe(hueFor("b"));
  });

  it("prefers a display name, then address, then key", () => {
    expect(threadTitle({ display_name: "Sam", sender_address: "@sam", thread_key: "k" })).toBe("Sam");
    expect(threadTitle({ sender_address: "@sam", thread_key: "k" })).toBe("@sam");
    expect(threadTitle({ thread_key: "k" })).toBe("k");
    expect(truncate("a  very long   sentence", 8)).toBe("a very…");
  });

  it("formats relative time and day boundaries", () => {
    const now = Date.parse("2026-09-01T12:00:00Z");
    expect(relativeTime(now - 3 * 60_000, now)).toMatch(/3 minutes ago/);
    expect(relativeTime(now - 2 * 3_600_000, now)).toMatch(/2 hours ago/);
    expect(relativeTime("not a date", now)).toBe("");
    expect(sameDay("2026-09-01T01:00:00", "2026-09-01T23:00:00")).toBe(true);
    expect(msUntil(now + 1000, now)).toBe(1000);
  });
});

describe("Region chrome", () => {
  it("renders a title row with actions and custom loading content", () => {
    const { rerender } = renderWithProviders(
      <Region surfaceId="y" state="loading" testId="region2" title="Waiting" actions={<button type="button">act</button>} loading={<p data-testid="custom-loading">…</p>}>
        <p>body</p>
      </Region>,
    );
    expect(screen.getByTestId("custom-loading")).toBeInTheDocument();
    expect(screen.getByRole("button")).toBeInTheDocument();
    rerender(
      <Region surfaceId="y" state="error" testId="region2" error={<p data-testid="custom-error">bad</p>}>
        <p>body</p>
      </Region>,
    );
    expect(screen.getByTestId("custom-error")).toBeInTheDocument();
    rerender(
      <Region surfaceId="y" state="ready" testId="region2" actions={<span>only actions</span>}>
        <p data-testid="ready-body-2">body</p>
      </Region>,
    );
    expect(screen.getByTestId("ready-body-2")).toBeInTheDocument();
    renderWithProviders(<TierBadge tier="owner" size="md" testId="tier-md" />);
    expect(screen.getByTestId("tier-md")).toHaveAttribute("data-tier", "owner");
    renderWithProviders(<AgentMark name="x" size="xs" />);
    renderWithProviders(<ChannelChip id="slack" size="md" />);
    expect(screen.getByRole("img", { name: "slack" })).toBeInTheDocument();
  });
});
