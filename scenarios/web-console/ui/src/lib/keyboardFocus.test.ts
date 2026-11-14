import { act } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { holdKeyboardForNextField } from "./keyboardFocus";

describe("holdKeyboardForNextField", () => {
  afterEach(() => vi.useRealTimers());

  it("focuses an invisible stand-in and removes it when focus leaves", () => {
    holdKeyboardForNextField();
    const standIn = document.body.querySelector('input[aria-hidden="true"]');
    expect(standIn).toBeInstanceOf(HTMLInputElement);
    expect(document.activeElement).toBe(standIn);
    (standIn as HTMLInputElement).blur();
    expect(document.body.contains(standIn)).toBe(false);
  });

  it("lowers the keyboard after the grace period if no field takes focus", () => {
    vi.useFakeTimers();
    holdKeyboardForNextField();
    const standIn = document.body.querySelector('input[aria-hidden="true"]') as HTMLInputElement;
    act(() => { vi.advanceTimersByTime(1000); });
    expect(document.body.contains(standIn)).toBe(false);
  });
});
