import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ getCurrentSession: vi.fn(), startFocus: vi.fn(), pauseFocus: vi.fn(), resumeFocus: vi.fn(), endFocus: vi.fn(), listActuals: vi.fn(), listActualCorrections: vi.fn(), recordManualActual: vi.fn(), correctActual: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { correctActual, endFocus, fetchActualCorrections, fetchActuals, fetchCurrentFocus, fetchPauseReasons, fetchSessionNotes, pauseFocus, recordManualActual, recordPauseReason, resumeFocus, saveSessionNote, startFocus } from "./focus";

afterEach(() => vi.clearAllMocks());

describe("focus API transport", () => {
  it("maps current and lifecycle calls through Connect", async () => {
    const session = { id: "s-1", revision: 1n } as never;
    client.getCurrentSession.mockResolvedValue({ hasSession: true, session });
    client.startFocus.mockResolvedValue({ session });
    client.pauseFocus.mockResolvedValue({ session });
    client.resumeFocus.mockResolvedValue({ session });
    client.endFocus.mockResolvedValue({ session });
    expect(await fetchCurrentFocus()).toBe(session);
    expect(await startFocus({ workItemId: "w-1", title: "Draft", mode: "open" })).toBe(session);
    await pauseFocus(session); await resumeFocus(session); await endFocus(session);
    expect(client.pauseFocus).toHaveBeenCalledWith({ sessionId: "s-1", expectedRevision: 1n });
    expect(client.resumeFocus).toHaveBeenCalledWith({ sessionId: "s-1", expectedRevision: 1n });
    expect(client.endFocus).toHaveBeenCalledWith({ sessionId: "s-1", expectedRevision: 1n });
  });

  it("returns no current session honestly", async () => {
    client.getCurrentSession.mockResolvedValue({ hasSession: false });
    expect(await fetchCurrentFocus()).toBeNull();
  });

  it("records, lists, and corrects manual actuals through Connect", async () => {
    const actual = { id: "a-1", localDate: "2026-09-19", reportedMinutes: 45n, certainty: "user_reported_approximate", revision: 1n } as never;
    client.listActuals.mockResolvedValue({ actuals: [actual] });
    client.listActualCorrections.mockResolvedValue({ corrections: [{ id: "c-1", actualId: "a-1" }] });
    client.recordManualActual.mockResolvedValue({ actual });
    client.correctActual.mockResolvedValue({ actual });
    expect(await fetchActuals("2026-09-19")).toEqual([actual]);
    expect(await fetchActualCorrections({ localDate: "2026-09-19" })).toEqual([{ id: "c-1", actualId: "a-1" }]);
    expect(await recordManualActual({ title: "Review", localDate: "2026-09-19", reportedMinutes: 45 })).toBe(actual);
    expect(await correctActual(actual, { reportedMinutes: 30, note: "Interrupted" })).toBe(actual);
    expect(client.recordManualActual).toHaveBeenCalledWith(expect.objectContaining({ reportedMinutes: 45n }));
    expect(client.correctActual).toHaveBeenCalledWith(expect.objectContaining({ id: "a-1", expectedRevision: 1n, reportedMinutes: 30n }));
  });

  it("persists and reads end-of-session notes through the REST seam", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ session_id: "s-1", local_date: "2026-09-19", note: "Shipped the outline", updated_at: "2026-09-19T10:00:00Z" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ session_id: "s-1", local_date: "2026-09-19", note: "Shipped the outline", updated_at: "2026-09-19T10:00:00Z" }]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(saveSessionNote({ sessionId: "s/1", note: "Shipped the outline" })).resolves.toMatchObject({ sessionId: "s-1", note: "Shipped the outline" });
    await expect(fetchSessionNotes("2026-09-19")).resolves.toEqual([{ sessionId: "s-1", localDate: "2026-09-19", note: "Shipped the outline", updatedAt: "2026-09-19T10:00:00Z" }]);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("persists and reads pause reasons through the REST seam", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ id: "p-1", session_id: "s-1", local_date: "2026-09-19", reason: "distracted", recorded_at: "2026-09-19T10:00:00Z" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ id: "p-1", session_id: "s-1", local_date: "2026-09-19", reason: "distracted", recorded_at: "2026-09-19T10:00:00Z" }]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(recordPauseReason({ sessionId: "s/1", reason: "distracted" })).resolves.toMatchObject({ sessionId: "s-1", reason: "distracted" });
    await expect(fetchPauseReasons("2026-09-19")).resolves.toEqual([{ id: "p-1", sessionId: "s-1", localDate: "2026-09-19", reason: "distracted", recordedAt: "2026-09-19T10:00:00Z" }]);
  });
});
