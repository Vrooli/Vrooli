import { test } from 'node:test';
import assert from 'node:assert/strict';
import { observeVitestSyntax, selectedVitestRules } from './vitest-profile.mjs';

const prelude = 'import {test, expect} from "vitest";\n';
test('selected subset is advisory and excludes source assertion-presence heuristics', () => {
  assert.deepEqual(Object.keys(selectedVitestRules), ['vitest/no-focused-tests', 'vitest/valid-expect']);
  assert.equal(selectedVitestRules['vitest/no-focused-tests'], 'warn');
  assert.equal(selectedVitestRules['vitest/valid-expect'][0], 'warn');
  assert.equal(observeVitestSyntax(prelude+'test("known",()=>expect(2).toBe(2))').configDigest,'831500ad6837f6467c495f22adac68f188cdf86d40d6480a90147c3f3694b23e');
});
for (const [name, body, rule] of [
  ['bare expect', 'test("bare",()=>{expect(2)})', 'malformed-expectation'],
  ['focus', 'test.only("focus",()=>{expect(2).toBe(2)})', 'focused-test'],
  ['missing await', 'test("async",()=>{expect(Promise.resolve(2)).resolves.toBe(2)})', 'async-assertion'],
]) test(name, () => {
  const report = observeVitestSyntax(prelude + body);
  assert.equal(report.status, 'observed');
  assert.equal(report.diagnostics.length, 1);
  assert.equal(report.diagnostics[0].canonicalRule, rule);
  assert.equal(report.diagnostics[0].line, 2);
  assert.ok(report.diagnostics[0].column > 0);
  assert.equal(report.diagnostics[0].severity, 1);
});
for (const [name, source] of [
  ['returned promise', prelude+'test("return",()=>expect(Promise.resolve(2)).resolves.toBe(2))'],
  ['awaited promise', prelude+'test("await",async()=>{await expect(Promise.resolve(2)).resolves.toBe(2)})'],
  ['custom matcher', prelude+'test("custom",()=>{expect(2).toBeEven()})'],
  ['lookalike string', prelude+'test("literal",()=>{expect("test.only( fake )").toContain("fake")})'],
  ['alias import', 'import {test as check, expect as verify} from "vitest"; check("alias",()=>verify(2).toBe(2))'],
  ['extended test', prelude+'const check = test.extend({answer:2}); check("extended",({answer})=>expect(answer).toBe(2))'],
]) test(name, () => {
  const report = observeVitestSyntax(source);
  assert.equal(report.status, 'observed');
  assert.deepEqual(report.diagnostics, []);
});
test('malformed source remains unknown, not clean', () => {
  const report=observeVitestSyntax('import { test from "vitest";');
  assert.equal(report.status,'unknown');
  assert.equal(report.reason,'parse-failure');
});
test('unsupported focus aliases remain unknown while supported alias assertions are checked', () => {
  const alias=observeVitestSyntax('import {test as check, expect as verify} from "vitest"; check.only("alias",()=>{verify(2)})');
  assert.equal(alias.checks.find(c=>c.rule==='focused-test').status,'unknown');
  assert.equal(alias.checks.find(c=>c.rule==='malformed-expectation').status,'violation');
  const extended=observeVitestSyntax(prelude+'const check=test.extend({answer:2}); check("extended",({answer})=>{expect(answer)})');
  assert.deepEqual(extended.diagnostics.map(d=>d.canonicalRule),['malformed-expectation']);
  assert.equal(extended.checks.find(c=>c.rule==='focused-test').status,'unknown');
});
test('local test lookalikes never become focus violations in canonical checks', () => {
  const report=observeVitestSyntax('const test={only:()=>{}}; test.only();');
  assert.equal(report.checks.find(c=>c.rule==='focused-test').status,'unknown');
});
test('TSX source is parsed by both native lint and scope applicability', () => {
  const report=observeVitestSyntax(prelude+'test("jsx",()=>{expect(<button>Save</button>).toBeTruthy()})','component.test.tsx');
  assert.equal(report.status,'observed');
  assert.deepEqual(report.diagnostics,[]);
});
for (const source of ['test["only"]("computed",()=>expect(2).toBe(2))','test.only.each([2])("each",n=>expect(n).toBe(2))','test.concurrent.only("concurrent",()=>expect(2).toBe(2))']) {
  test('unsupported focused call shape remains unknown: '+source, () => {
    assert.equal(observeVitestSyntax(prelude+source).checks.find(c=>c.rule==='focused-test').status,'unknown');
  });
}
