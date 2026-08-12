import { describe, it, expect, vi } from "vitest";
import { sendComboSequence } from "../lib/comboSequence";
describe("sendComboSequence", () => {
    it("single-step combo calls onInput once", async () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        const fakeDelay = vi.fn(() => Promise.resolve());
        const steps = [{ data: "\x03" }];
        await sendComboSequence(steps, onInput, fakeDelay);
        expect(onInput).toHaveBeenCalledTimes(1);
        expect(onInput).toHaveBeenCalledWith("\x03", "toolbar-key");
        expect(fakeDelay).not.toHaveBeenCalled();
    });
    it("multi-step combo calls onInput in order with correct data", async () => {
        const calls = [];
        const onInput = vi.fn((data) => { calls.push(data); return { status: "sent", seq: 1 }; });
        const fakeDelay = vi.fn(() => Promise.resolve());
        const steps = [
            { data: "\x03" },
            { data: "\x03", delayMs: 80 },
        ];
        await sendComboSequence(steps, onInput, fakeDelay);
        expect(calls).toEqual(["\x03", "\x03"]);
        expect(onInput).toHaveBeenCalledTimes(2);
    });
    it("calls delay with correct delayMs values", async () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        const fakeDelay = vi.fn(() => Promise.resolve());
        const steps = [
            { data: "\x03" },
            { data: "\x04", delayMs: 100 },
            { data: "\x1a", delayMs: 50 },
        ];
        await sendComboSequence(steps, onInput, fakeDelay);
        expect(fakeDelay).toHaveBeenCalledTimes(2);
        expect(fakeDelay).toHaveBeenCalledWith(100);
        expect(fakeDelay).toHaveBeenCalledWith(50);
    });
    it("skips delay for steps without delayMs or delayMs: 0", async () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        const fakeDelay = vi.fn(() => Promise.resolve());
        const steps = [
            { data: "\x03" },
            { data: "\x04", delayMs: 0 },
            { data: "\x1a" },
        ];
        await sendComboSequence(steps, onInput, fakeDelay);
        expect(fakeDelay).not.toHaveBeenCalled();
        expect(onInput).toHaveBeenCalledTimes(3);
    });
});
