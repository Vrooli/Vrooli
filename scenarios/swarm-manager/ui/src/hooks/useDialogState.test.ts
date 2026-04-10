import { renderHook, act } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useDialogState } from "./useDialogState";

interface TestEntity {
  id: string;
  name: string;
}

describe("useDialogState", () => {
  it("starts closed with create mode and null editing", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());
    expect(result.current.isOpen).toBe(false);
    expect(result.current.mode).toBe("create");
    expect(result.current.editing).toBeNull();
  });

  it("openCreate opens in create mode with null editing", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());

    act(() => result.current.openCreate());

    expect(result.current.isOpen).toBe(true);
    expect(result.current.mode).toBe("create");
    expect(result.current.editing).toBeNull();
  });

  it("openEdit opens in edit mode with the given entity", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());
    const entity = { id: "1", name: "Test" };

    act(() => result.current.openEdit(entity));

    expect(result.current.isOpen).toBe(true);
    expect(result.current.mode).toBe("edit");
    expect(result.current.editing).toEqual(entity);
  });

  it("close resets to initial state", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());

    act(() => result.current.openEdit({ id: "1", name: "Test" }));
    expect(result.current.isOpen).toBe(true);

    act(() => result.current.close());

    expect(result.current.isOpen).toBe(false);
    expect(result.current.mode).toBe("create");
    expect(result.current.editing).toBeNull();
  });

  it("transitions from edit to create correctly", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());

    act(() => result.current.openEdit({ id: "1", name: "First" }));
    expect(result.current.mode).toBe("edit");

    act(() => result.current.close());
    act(() => result.current.openCreate());

    expect(result.current.isOpen).toBe(true);
    expect(result.current.mode).toBe("create");
    expect(result.current.editing).toBeNull();
  });

  it("transitions from create to edit correctly", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());

    act(() => result.current.openCreate());
    expect(result.current.mode).toBe("create");

    // Directly switch to edit without closing first
    act(() => result.current.openEdit({ id: "2", name: "Second" }));

    expect(result.current.isOpen).toBe(true);
    expect(result.current.mode).toBe("edit");
    expect(result.current.editing).toEqual({ id: "2", name: "Second" });
  });

  it("openEdit replaces previous editing entity", () => {
    const { result } = renderHook(() => useDialogState<TestEntity>());

    act(() => result.current.openEdit({ id: "1", name: "First" }));
    act(() => result.current.openEdit({ id: "2", name: "Second" }));

    expect(result.current.editing).toEqual({ id: "2", name: "Second" });
  });

  it("callback references are stable across renders", () => {
    const { result, rerender } = renderHook(() => useDialogState<TestEntity>());

    const firstOpenCreate = result.current.openCreate;
    const firstOpenEdit = result.current.openEdit;
    const firstClose = result.current.close;

    rerender();

    expect(result.current.openCreate).toBe(firstOpenCreate);
    expect(result.current.openEdit).toBe(firstOpenEdit);
    expect(result.current.close).toBe(firstClose);
  });

  it("works with primitive editing type (string)", () => {
    const { result } = renderHook(() => useDialogState<string>());

    act(() => result.current.openEdit("module-123"));

    expect(result.current.isOpen).toBe(true);
    expect(result.current.mode).toBe("edit");
    expect(result.current.editing).toBe("module-123");
  });
});
