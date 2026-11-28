import { create } from "@bufbuild/protobuf";
import { AskSchema, ListAsksRequest_Order } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { describe, expect, it } from "vitest";

import { attentionTier, closingHours, filterCount, groupByAttentionAndFamily, groupByFamily, listViewRequest, readListView, writeListView } from "./askListView";

const params = (query: string) => new URLSearchParams(query);
const ask = (state: string, urgency: string, closingSoon = false) => create(AskSchema, { id: `${state}-${urgency}${closingSoon ? "-closing" : ""}`, state, urgency, closingSoon });

describe("readListView", () => {
  it("opens Needs you on what needs the owner first and History on the newest", () => {
    expect(readListView(params(""), true)).toEqual({ sort: "attention", urgencies: [], senders: [] });
    expect(readListView(params(""), false)).toEqual({ sort: "newest", urgencies: [], senders: [] });
  });

  it("reads a sort, urgencies and senders from the address", () => {
    expect(readListView(params("sort=due&urgency=high&urgency=critical&sender=Research%20Desk&sender=Treasury"), true)).toEqual({
      sort: "due", urgencies: ["critical", "high"], senders: ["Research Desk", "Treasury"],
    });
  });

  it("ignores what it does not know instead of sending it to the server", () => {
    expect(readListView(params("sort=random&urgency=urgent&urgency=low&sender=&sender=%20%20&sender=A&sender=A&sender=" + "x".repeat(61)), false))
      .toEqual({ sort: "newest", urgencies: ["low"], senders: ["A"] });
  });
});

describe("writeListView", () => {
  it("leaves the default sort out of the address and keeps every other parameter", () => {
    const view = { sort: "attention" as const, urgencies: ["high" as const], senders: ["Treasury"] };
    expect(writeListView(params("q=disk&state=answered&sort=due"), view, true).toString()).toBe("q=disk&state=answered&urgency=high&sender=Treasury");
    expect(writeListView(params(""), { ...view, sort: "due" }, true).toString()).toBe("sort=due&urgency=high&sender=Treasury");
    expect(writeListView(params(""), { ...view, sort: "attention" }, false).toString()).toBe("sort=attention&urgency=high&sender=Treasury");
  });

  it("round-trips a view, including a sender that needs escaping", () => {
    const view = { sort: "oldest" as const, urgencies: ["critical" as const, "low" as const], senders: ["Ops & Infra", "Café"] };
    expect(readListView(writeListView(params("q=x"), view, false), false)).toEqual(view);
  });

  it("clears the filters when the view has none", () => {
    expect(writeListView(params("urgency=high&sender=A&sort=due"), { sort: "newest", urgencies: [], senders: [] }, false).toString()).toBe("");
  });
});

describe("listViewRequest", () => {
  it("always names the order for the queue and leaves History's default to the server", () => {
    expect(listViewRequest(readListView(params(""), true), true)).toEqual({ order: ListAsksRequest_Order.ATTENTION });
    expect(listViewRequest(readListView(params(""), false), false)).toEqual({});
    expect(listViewRequest(readListView(params("sort=oldest"), false), false)).toEqual({ order: ListAsksRequest_Order.OLDEST });
    expect(listViewRequest(readListView(params("sort=newest"), true), true)).toEqual({ order: ListAsksRequest_Order.NEWEST });
  });

  it("sends only the filters that are chosen", () => {
    expect(listViewRequest(readListView(params("sort=due&urgency=high&sender=Treasury"), true), true)).toEqual({
      order: ListAsksRequest_Order.DUE, urgencies: ["high"], senders: ["Treasury"],
    });
  });

  it("counts the filters in force", () => {
    expect(filterCount(readListView(params("urgency=high&urgency=low&sender=A"), true))).toBe(3);
  });
});

describe("attention groups", () => {
  it("puts an escalated ask above its urgency and names an unknown urgency normal", () => {
    expect(attentionTier(ask("escalated", "low"))).toBe("escalated");
    expect(attentionTier(ask("pending", "critical"))).toBe("critical");
    expect(attentionTier(ask("pending", ""))).toBe("normal");
  });

  it("puts an ask the Hub marked closing soon above its urgency, and below an escalated one", () => {
    expect(attentionTier(ask("pending", "low", true))).toBe("closing");
    expect(attentionTier(ask("escalated", "low", true))).toBe("escalated");
    const groups = groupByAttentionAndFamily([ask("escalated", "normal"), ask("pending", "low", true), ask("pending", "normal", true), ask("pending", "critical")], true);
    expect(groups.map((group) => [group.tier, group.families.reduce((count, family) => count + family.asks.length, 0)])).toEqual([["escalated", 1], ["closing", 2], ["critical", 1]]);
  });

  it("reads the window in whole hours", () => {
    expect(closingHours(6 * 3600)).toBe(6);
    expect(closingHours(undefined)).toBe(0);
  });

  it("groups consecutive asks of one tier, in the order they came", () => {
    const groups = groupByAttentionAndFamily([ask("escalated", "normal"), ask("pending", "critical"), ask("pending", "critical"), ask("pending", "normal")], true);
    expect(groups.map((group) => [group.tier, group.families.flatMap((family) => family.asks.map((a) => a.id))])).toEqual([
      ["escalated", ["escalated-normal"]],
      ["critical", ["pending-critical", "pending-critical"]],
      ["normal", ["pending-normal"]],
    ]);
  });
});

describe("decision families", () => {
  it("shows three guard checks as one cause group while retaining each answerable ask", () => {
    const checks = ["network", "backup", "permissions"].map((id) => create(AskSchema, {
      id,
      question: `May the ${id} guard pass?`,
      decisionFamily: "release-guards",
      decisionCause: "The release is waiting on its three independent safety checks.",
      decisionKind: "review",
    }));
    const groups = groupByFamily(checks);
    expect(groups).toHaveLength(1);
    expect(groups[0]).toMatchObject({ key: "release-guards", cause: "The release is waiting on its three independent safety checks." });
    expect(groups[0]!.asks.map((item) => item.id)).toEqual(["network", "backup", "permissions"]);
  });

  it("keeps a family together when its checks have different attention tiers", () => {
    const family = [
      create(AskSchema, { id: "urgent", state: "escalated", decisionFamily: "release", decisionCause: "Same release cause" }),
      create(AskSchema, { id: "routine", state: "pending", decisionFamily: "release", decisionCause: "Same release cause" }),
    ];
    const groups = groupByAttentionAndFamily(family, true);
    expect(groups).toHaveLength(1);
    expect(groups[0]!.tier).toBe("escalated");
    expect(groups[0]!.families[0]!.asks.map((item) => item.id)).toEqual(["urgent", "routine"]);
  });

  it("keeps asks without a complete family isolated and does not infer grouping from sender", () => {
    const first = create(AskSchema, { id: "one", sender: "same", decisionFamily: "release" });
    const second = create(AskSchema, { id: "two", sender: "same", decisionCause: "same reason" });
    const groups = groupByFamily([first, second]);
    expect(groups.map((group) => group.asks.map((item) => item.id))).toEqual([["one"], ["two"]]);
  });

  it("keeps distinct causes separate even when a producer reuses a family key", () => {
    const first = create(AskSchema, { id: "one", decisionFamily: "release", decisionCause: "Cause A" });
    const second = create(AskSchema, { id: "two", decisionFamily: "release", decisionCause: "Cause B" });
    expect(groupByFamily([first, second]).map((group) => group.cause)).toEqual(["Cause A", "Cause B"]);
  });
});
