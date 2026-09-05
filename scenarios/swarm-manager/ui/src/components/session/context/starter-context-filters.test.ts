import { describe, expect, it } from "vitest";
import {
  EXECUTION_STALE_MS,
  backlogItemIsStale,
  countForStarterCard,
  executionIsFailedOrStale,
} from "./starter-context-filters";
import type { AgentSessionContextType, BacklogItem, ExecutionRecord } from "../../../types";
import type { SessionContextOption } from "./session-context-refs";

const NOW = new Date("2026-05-31T12:00:00Z").getTime();

function exec(status: string, updatedAtMsAgo: number): ExecutionRecord {
  const updatedAt = new Date(NOW - updatedAtMsAgo).toISOString();
  return {
    executionId: `${status}-${updatedAtMsAgo}`,
    backlogKind: "fix",
    backlogName: "demo",
    status,
    createdAt: updatedAt,
    updatedAt,
  } as unknown as ExecutionRecord;
}

describe("executionIsFailedOrStale", () => {
  it("treats failed and canceled as actionable regardless of age", () => {
    expect(executionIsFailedOrStale(exec("failed", 0), NOW)).toBe(true);
    expect(executionIsFailedOrStale(exec("canceled", 0), NOW)).toBe(true);
  });

  it("never flags healthy terminal completions", () => {
    expect(executionIsFailedOrStale(exec("completed", 10 * EXECUTION_STALE_MS), NOW)).toBe(false);
  });

  it("flags a non-terminal run only once it goes stale", () => {
    expect(executionIsFailedOrStale(exec("running", EXECUTION_STALE_MS - 1), NOW)).toBe(false);
    expect(executionIsFailedOrStale(exec("running", EXECUTION_STALE_MS + 1), NOW)).toBe(true);
    expect(executionIsFailedOrStale(exec("needs_review", EXECUTION_STALE_MS + 1), NOW)).toBe(true);
  });
});

describe("countForStarterCard", () => {
  const optionsByType = {
    backlog_item: [{} as SessionContextOption, {} as SessionContextOption, {} as SessionContextOption],
    execution: [{}, {}, {}, {}] as SessionContextOption[],
  } as Record<AgentSessionContextType, SessionContextOption[]>;

  it("uses the full type list length when no filter key is set", () => {
    expect(countForStarterCard({ optionsByType, executions: [], type: "backlog_item" })).toBe(3);
  });

  it("narrows to the actionable subset for the failed/stale execution filter", () => {
    const executions = [
      exec("failed", 0),
      exec("completed", 0),
      exec("running", EXECUTION_STALE_MS + 1), // stale
      exec("running", 0), // fresh, healthy
    ];
    expect(
      countForStarterCard({
        optionsByType,
        executions,
        type: "execution",
        filterKey: "execution_failed_or_stale",
        now: NOW,
      }),
    ).toBe(2);
  });
});

describe("backlogItemIsStale", () => {
  const item = (stale?: boolean) => ({ kind: "fix", name: "demo", stale } as unknown as BacklogItem);

  it("takes the server's staleness verdict verbatim", () => {
    // The server folds update age, last accepted review, plan-ref integrity,
    // and acceptance-path validity into this flag. Re-deriving any of it here
    // would give the operator a second definition that quietly disagrees.
    expect(backlogItemIsStale(item(true))).toBe(true);
    expect(backlogItemIsStale(item(false))).toBe(false);
  });

  it("treats an unprojected flag as not stale rather than guessing", () => {
    expect(backlogItemIsStale(item(undefined))).toBe(false);
  });

  it("counts only stale items for a stale-filtered card", () => {
    expect(
      countForStarterCard({
        optionsByType: {} as Record<AgentSessionContextType, SessionContextOption[]>,
        executions: [],
        backlogItems: [item(true), item(false), item(true), item(undefined)],
        type: "backlog_item",
        filterKey: "backlog_item_stale",
      }),
    ).toBe(2);
  });
});
