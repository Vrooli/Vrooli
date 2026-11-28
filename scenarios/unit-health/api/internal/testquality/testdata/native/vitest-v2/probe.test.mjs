import { test, expect, describe, beforeEach } from "vitest";
import assert from "node:assert/strict";

test("C021 direct matcher", () => expect(2 + 3).toBe(5));
test("C023 empty", () => {});
test("C024 unreachable", () => { if (false) expect(2 + 3).toBe(5); });
test("C025 bare expect", () => { expect(2 + 3); });
test("C026 tautology", () => expect(true).toBe(true));
describe("C027 hook", () => {
  beforeEach(() => expect(2 + 3).toBe(5));
  test("empty body", () => {});
});
test("C028 Node assertion", () => assert.equal(2 + 3, 5));
test.each([1, 2])("C029 parameter %s", value => {
  if (value === 1) expect(value).toBe(1);
});
test.concurrent("C030 local expect first", async ({ expect: localExpect }) => {
  await Promise.resolve();
  localExpect(2 + 3).toBe(5);
});
test.concurrent("C030 local expect second", async ({ expect: localExpect }) => {
  await Promise.resolve();
  localExpect(3 + 4).toBe(7);
});
expect.extend({ toBeFive(value) {
  return { pass: value === 5, message: () => `expected ${value} to be five` };
} });
test("C031 failing custom matcher", () => expect(4).toBeFive());
describe("C032 setup", () => {
  beforeEach(() => { throw new Error("fixture setup failure"); });
  test("body never executes", () => expect(2 + 3).toBe(5));
});
let retryAttempt = 0;
test("C033 retry", { retry: 1 }, () => {
  retryAttempt++;
  expect(retryAttempt).toBe(2);
});
test.skip("C034 skipped", () => {});
const extended = test.extend({ answer: async ({}, use) => { await use(5); } });
extended("C037 extended", ({ answer }) => expect(answer).toBe(5));
test("C039 returned async matcher", () => expect(Promise.resolve(5)).resolves.toBe(5));
