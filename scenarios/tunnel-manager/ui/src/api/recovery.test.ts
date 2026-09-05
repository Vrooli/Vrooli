import { describe, expect, it, vi } from "vitest";

import { EventOutcome, getState, listEvents, recover, recoveryClient } from "./recovery";
import { makeRecoveryEvent, makeRecoveryState } from "../test-utils/mocks/recovery";

describe("recovery API helpers", () => {
  it("returns the state snapshot from the generated client", async () => {
    const state = makeRecoveryState({ consecFailures: 3 });
    const spy = vi.spyOn(recoveryClient, "getState").mockResolvedValueOnce({ state } as never);

    await expect(getState()).resolves.toBe(state);
    expect(spy).toHaveBeenCalledWith({});
  });

  it("passes the requested event limit", async () => {
    const events = [makeRecoveryEvent({ id: "event-1" }), makeRecoveryEvent({ id: "event-2" })];
    const spy = vi.spyOn(recoveryClient, "listEvents").mockResolvedValueOnce({ events } as never);

    await expect(listEvents(2)).resolves.toBe(events);
    expect(spy).toHaveBeenCalledWith({ limit: 2 });
  });

  it("returns manual recovery outcome and event", async () => {
    const event = makeRecoveryEvent({ trigger: "manual" });
    const spy = vi
      .spyOn(recoveryClient, "recover")
      .mockResolvedValueOnce({ outcome: EventOutcome.SUCCESS, event } as never);

    await expect(recover(true)).resolves.toEqual({ outcome: EventOutcome.SUCCESS, event });
    expect(spy).toHaveBeenCalledWith({ force: true });
  });
});
