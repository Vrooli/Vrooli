import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listTodayAllocations: vi.fn(), listAllocations: vi.fn(), createAllocation: vi.fn(), previewAllocation: vi.fn(), applyAllocationProposal: vi.fn(), previewSchedule: vi.fn(), applyScheduleProposal: vi.fn(), carryForwardAllocation: vi.fn(), listRoutines: vi.fn(), createRoutine: vi.fn(), listRoutineOccurrences: vi.fn(), skipRoutineOccurrence: vi.fn(), rescheduleRoutineOccurrence: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { applyAllocationProposal, applyScheduleProposal, carryForwardAllocation, createAllocation, createRoutine, fetchAllocations, fetchRoutineOccurrences, fetchRoutines, fetchTodayAllocations, previewAllocation, previewSchedule, rescheduleRoutineOccurrence, skipRoutineOccurrence } from "./calendar";

afterEach(() => vi.clearAllMocks());

describe("calendar API transport", () => {
  it("reads accepted allocations", async () => {
    const response = { plannedMinutes: 45, allocations: [] } as never;
    client.listTodayAllocations.mockResolvedValueOnce(response);
    expect(await fetchTodayAllocations()).toBe(response);
    expect(client.listTodayAllocations).toHaveBeenCalledWith({ localDate: "" });
  });

  it("reads a target day's capacity and carries one allocation", async () => {
    const response = { availableMinutes: 360, breathingRoomMinutes: 315, allocations: [] } as never;
    const allocation = { id: "a-2", carriedFromId: "a-1" } as never;
    client.listTodayAllocations.mockResolvedValueOnce(response);
    client.carryForwardAllocation.mockResolvedValueOnce({ allocation });
    expect(await fetchTodayAllocations("2026-09-22")).toBe(response);
    expect(client.listTodayAllocations).toHaveBeenCalledWith({ localDate: "2026-09-22" });
    expect(await carryForwardAllocation({ allocationId: "a-1", targetLocalDate: "2026-09-22", startMinutes: 660 })).toBe(allocation);
    expect(client.carryForwardAllocation).toHaveBeenCalledWith({ allocationId: "a-1", targetLocalDate: "2026-09-22", startMinutes: 660 });
    client.carryForwardAllocation.mockResolvedValueOnce({});
    await expect(carryForwardAllocation({ allocationId: "a-1", targetLocalDate: "2026-09-22", startMinutes: 660 })).rejects.toThrow("carried allocation was not returned");
  });

  it("reads an explicit local-date range", async () => {
    const response = { allocations: [] } as never;
    client.listAllocations.mockResolvedValueOnce(response);
    expect(await fetchAllocations("2026-09-21", "2026-09-27")).toBe(response);
    expect(client.listAllocations).toHaveBeenCalledWith({ startLocalDate: "2026-09-21", endLocalDate: "2026-09-27" });
  });

  it("creates an allocation and rejects an empty response", async () => {
    const allocation = { id: "a-1" } as never;
    client.createAllocation.mockResolvedValueOnce({ allocation });
    expect(await createAllocation({ workItemId: "w-1", localDate: "2026-09-19", startMinutes: 600, durationMinutes: 45 })).toBe(allocation);
    expect(client.createAllocation).toHaveBeenCalledWith({ workItemId: "w-1", localDate: "2026-09-19", startMinutes: 600, durationMinutes: 45 });
    client.createAllocation.mockResolvedValueOnce({});
    await expect(createAllocation({ workItemId: "w-1", localDate: "2026-09-19", startMinutes: 600, durationMinutes: 45 })).rejects.toThrow("allocation was not returned");
  });

  it("previews a placement and rejects an empty proposal", async () => {
    const proposal = { state: "feasible", startMinutes: 615, durationMinutes: 45, reason: "Moved" } as never;
    client.previewAllocation.mockResolvedValueOnce({ proposal });
    expect(await previewAllocation({ workItemId: "w-1", localDate: "2026-09-19", startMinutes: 600, durationMinutes: 45 })).toBe(proposal);
    expect(client.previewAllocation).toHaveBeenCalledWith({ workItemId: "w-1", localDate: "2026-09-19", startMinutes: 600, durationMinutes: 45 });
    client.previewAllocation.mockResolvedValueOnce({});
    await expect(previewAllocation({ workItemId: "w-1", localDate: "2026-09-19", startMinutes: 600, durationMinutes: 45 })).rejects.toThrow("placement proposal was not returned");
  });

  it("applies a proposal and rejects an empty allocation", async () => {
    const allocation = { id: "a-1" } as never;
    client.applyAllocationProposal.mockResolvedValueOnce({ allocation });
    expect(await applyAllocationProposal({ proposalId: "p-1", expectedRevision: 1n, idempotencyKey: "p-1:apply" })).toBe(allocation);
    expect(client.applyAllocationProposal).toHaveBeenCalledWith({ proposalId: "p-1", expectedRevision: 1n, idempotencyKey: "p-1:apply" });
    client.applyAllocationProposal.mockResolvedValueOnce({});
    await expect(applyAllocationProposal({ proposalId: "p-1", expectedRevision: 1n, idempotencyKey: "p-1:apply" })).rejects.toThrow("applied allocation was not returned");
  });

  it("previews and applies a multi-item schedule proposal", async () => {
    const proposal = { id: "sp-1", state: "partial", baseRevision: 2n, placements: [{ workItemId: "w-1", state: "feasible" }] } as never;
    const allocations = [{ id: "a-1" }] as never;
    client.previewSchedule.mockResolvedValueOnce({ proposal });
    client.applyScheduleProposal.mockResolvedValueOnce({ allocations });
    expect(await previewSchedule({ localDate: "2026-09-19", startMinutes: 540, workItemIds: ["w-1", "w-2"] })).toBe(proposal);
    expect(await applyScheduleProposal({ proposalId: "sp-1", expectedRevision: 2n, idempotencyKey: "sp-1:apply" })).toBe(allocations);
    expect(client.previewSchedule).toHaveBeenCalledWith({ localDate: "2026-09-19", startMinutes: 540, workItemIds: ["w-1", "w-2"] });
    expect(client.applyScheduleProposal).toHaveBeenCalledWith({ proposalId: "sp-1", expectedRevision: 2n, idempotencyKey: "sp-1:apply" });
    client.previewSchedule.mockResolvedValueOnce({});
    await expect(previewSchedule({ localDate: "2026-09-19", startMinutes: 540, workItemIds: ["w-1"] })).rejects.toThrow("schedule proposal was not returned");
  });

  it("reads, creates, and expands routines", async () => {
    const routine = { id: "routine-1", kind: "flexible" } as never;
    const occurrence = { routineId: "routine-1", localDate: "2026-09-21" } as never;
    client.listRoutines.mockResolvedValueOnce({ routines: [routine] });
    client.createRoutine.mockResolvedValueOnce({ routine });
    client.listRoutineOccurrences.mockResolvedValueOnce({ occurrences: [occurrence] });
    expect(await fetchRoutines()).toEqual([routine]);
    expect(await createRoutine({ title: "Runs", kind: "flexible", timezone: "UTC", startDate: "2026-09-21", endDate: "", weekdays: [1, 3, 5], startMinute: 540, durationMinutes: 30, frequencyPerWeek: 2 })).toBe(routine);
    expect(await fetchRoutineOccurrences("2026-09-21", "2026-09-27")).toEqual([occurrence]);
    expect(client.listRoutineOccurrences).toHaveBeenCalledWith({ startLocalDate: "2026-09-21", endLocalDate: "2026-09-27" });
  });

  it("skips one occurrence and rejects a false response", async () => {
    client.skipRoutineOccurrence.mockResolvedValueOnce({ skipped: true });
    await expect(skipRoutineOccurrence({ routineId: "routine-1", localDate: "2026-09-23", expectedRevision: 1n })).resolves.toBeUndefined();
    expect(client.skipRoutineOccurrence).toHaveBeenCalledWith({ routineId: "routine-1", localDate: "2026-09-23", expectedRevision: 1n });
    client.skipRoutineOccurrence.mockResolvedValueOnce({ skipped: false });
    await expect(skipRoutineOccurrence({ routineId: "routine-1", localDate: "2026-09-23", expectedRevision: 1n })).rejects.toThrow("not skipped");
  });

  it("reschedules one occurrence and rejects a false response", async () => {
    client.rescheduleRoutineOccurrence.mockResolvedValueOnce({ rescheduled: true });
    await expect(rescheduleRoutineOccurrence({ routineId: "routine-1", localDate: "2026-09-23", startMinute: 570, expectedRevision: 1n })).resolves.toBeUndefined();
    expect(client.rescheduleRoutineOccurrence).toHaveBeenCalledWith({ routineId: "routine-1", localDate: "2026-09-23", startMinute: 570, expectedRevision: 1n });
    client.rescheduleRoutineOccurrence.mockResolvedValueOnce({ rescheduled: false });
    await expect(rescheduleRoutineOccurrence({ routineId: "routine-1", localDate: "2026-09-23", startMinute: 570, expectedRevision: 1n })).rejects.toThrow("not rescheduled");
  });
});
