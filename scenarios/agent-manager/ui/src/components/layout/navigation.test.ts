import assert from "node:assert/strict";
import { test } from "vitest";
import { MAX_PHONE_TABS, NAV_DESTINATIONS, sectionForPath } from "./navigation.js";

test("the phone tab bar carries at most five destinations and every destination stays in the full list", () => {
  const phoneTabs = NAV_DESTINATIONS.filter((item) => item.phoneTab);
  assert.ok(phoneTabs.length <= MAX_PHONE_TABS, `phone tabs: ${phoneTabs.map((item) => item.id).join(", ")}`);
  assert.equal(MAX_PHONE_TABS, 5);
  assert.deepEqual(phoneTabs.map((item) => item.id), ["dashboard", "tasks", "runs", "usage"]);
  assert.equal(new Set(NAV_DESTINATIONS.map((item) => item.id)).size, NAV_DESTINATIONS.length);
});

test("desktop order keeps each group contiguous: work, then insight, then system", () => {
  const groups = NAV_DESTINATIONS.map((item) => item.group);
  const order = groups.filter((group, index) => groups.indexOf(group) === index);
  assert.deepEqual(order, ["work", "insight", "system"]);
  for (const group of order) {
    const first = groups.indexOf(group);
    const last = groups.lastIndexOf(group);
    assert.ok(groups.slice(first, last + 1).every((value) => value === group), `${group} is split`);
  }
});

test("Usage replaces Stats and Import as destinations", () => {
  const ids = NAV_DESTINATIONS.map((item) => item.id as string);
  assert.ok(ids.includes("usage"));
  assert.equal(ids.includes("stats"), false);
  assert.equal(ids.includes("import"), false);
});

test("sectionForPath maps nested routes to their destination and unknown paths to the dashboard", () => {
  assert.equal(sectionForPath("/"), "dashboard");
  assert.equal(sectionForPath("/runs/abc"), "runs");
  assert.equal(sectionForPath("/usage"), "usage");
  assert.equal(sectionForPath("/observability"), "health");
  assert.equal(sectionForPath("/runsx"), "dashboard");
  assert.equal(sectionForPath("/nowhere"), "dashboard");
});

test("Investigations is routable but not a destination; it belongs to Health", () => {
  assert.equal(NAV_DESTINATIONS.some((item) => (item.id as string) === "investigations"), false);
  assert.equal(sectionForPath("/investigations"), "health");
  assert.equal(sectionForPath("/investigations/abc"), "health");
});
