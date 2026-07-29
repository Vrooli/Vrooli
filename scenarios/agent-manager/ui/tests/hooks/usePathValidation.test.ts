import assert from "node:assert/strict";
import { act, renderHook } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useProjectRootValidation, useScopePathValidation } from "../../src/hooks/usePathValidation.js";

const { validatePath } = vi.hoisted(() => ({ validatePath: vi.fn() }));
vi.mock("../../src/lib/api.js", () => ({ validatePath }));

async function elapse() { await act(async () => { await vi.advanceTimersByTimeAsync(300); }); }

afterEach(() => { vi.useRealTimers(); vi.resetAllMocks(); });

test("useProjectRootValidation rejects unsafe client paths before the API", async () => {
  vi.useFakeTimers();
  const { result, rerender } = renderHook(({ path }) => useProjectRootValidation(path), { initialProps: { path: "relative" } });
  await elapse(); assert.deepEqual(result.current, { status: "invalid", message: "Path must be absolute (start with /)" });
  rerender({ path: "/tmp/" }); await elapse();
  assert.deepEqual(result.current, { status: "invalid", message: "Cannot use system directories as project root" });
  assert.equal(validatePath.mock.calls.length, 0);
});

test("useProjectRootValidation handles server success, failure, and unavailable API", async () => {
  vi.useFakeTimers(); validatePath.mockResolvedValueOnce({ valid: true });
  const { result, rerender } = renderHook(({ path }) => useProjectRootValidation(path), { initialProps: { path: "/work/project" } });
  await elapse(); assert.deepEqual(result.current, { status: "valid", message: "Path is valid" });
  validatePath.mockResolvedValueOnce({ valid: false, error: "Missing directory" }); rerender({ path: "/work/missing" }); await elapse();
  assert.deepEqual(result.current, { status: "invalid", message: "Missing directory" });
  validatePath.mockRejectedValueOnce(new Error("offline")); rerender({ path: "/work/offline" }); await elapse();
  assert.deepEqual(result.current, { status: "valid", message: "Path format is valid" });
});

test("useScopePathValidation handles relative, unsafe, outside, and server root checks", async () => {
  vi.useFakeTimers();
  const { result, rerender } = renderHook(({ path, root }) => useScopePathValidation(path, root), { initialProps: { path: "src", root: "/work/project" } });
  await elapse(); assert.deepEqual(result.current, { status: "valid", message: "Relative path" });
  rerender({ path: "/etc", root: "/work/project" }); await elapse(); assert.equal(result.current.status, "invalid");
  rerender({ path: "/other/file", root: "/work/project" }); await elapse(); assert.deepEqual(result.current, { status: "outside", message: "Must be within /work/project" });
  validatePath.mockResolvedValueOnce({ valid: false, withinProjectRoot: false, error: "Escapes root" }); rerender({ path: "/work/project/file", root: "/work/project" }); await elapse();
  assert.deepEqual(result.current, { status: "outside", message: "Escapes root" });
});
