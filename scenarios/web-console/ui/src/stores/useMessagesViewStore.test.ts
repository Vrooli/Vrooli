import { beforeEach, describe, expect, it } from "vitest";
import { MESSAGES_POSITION_TTL_MS, useMessagesViewStore } from "./useMessagesViewStore";

const KEY = "wc-messages-view";

function persisted(): { version: number; state: { viewModes: Record<string, string>; positions: Record<string, unknown> } } {
  return JSON.parse(localStorage.getItem(KEY) ?? "{}") as ReturnType<typeof persisted>;
}

const place = { topEventId: "e42", topSequence: 42, offsetPx: 18, follow: false };

describe("useMessagesViewStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
  });

  it("[REQ:P0-017b] persists the per-session view mode under wc-messages-view", () => {
    useMessagesViewStore.getState().setViewMode("s1", "messages");

    expect(persisted().version).toBe(1);
    expect(persisted().state.viewModes).toEqual({ s1: "messages" });
  });

  it("[REQ:P0-017b] stores the top message, its offset, follow, and the save time", () => {
    const before = Date.now();
    useMessagesViewStore.getState().savePosition("s1", place);

    const saved = useMessagesViewStore.getState().positions.s1;
    expect(saved).toMatchObject(place);
    expect(saved?.savedAt).toBeGreaterThanOrEqual(before);
    expect(persisted().state.positions.s1).toMatchObject(place);
  });

  it("[REQ:P0-017b] forget removes both the view mode and the position", () => {
    const store = useMessagesViewStore.getState();
    store.setViewMode("s1", "messages");
    store.savePosition("s1", place);
    store.setViewMode("s2", "messages");

    store.forget("s1");

    expect(useMessagesViewStore.getState().viewModes).toEqual({ s2: "messages" });
    expect(useMessagesViewStore.getState().positions).toEqual({});
  });

  it("[REQ:P0-017b] drops positions older than 30 days when it hydrates", async () => {
    const now = Date.now();
    localStorage.setItem(KEY, JSON.stringify({
      version: 1,
      state: {
        viewModes: { fresh: "messages" },
        positions: {
          fresh: { ...place, savedAt: now - 1000 },
          stale: { ...place, savedAt: now - MESSAGES_POSITION_TTL_MS - 1000 },
        },
      },
    }));

    await useMessagesViewStore.persist.rehydrate();

    expect(Object.keys(useMessagesViewStore.getState().positions)).toEqual(["fresh"]);
    expect(useMessagesViewStore.getState().viewModes).toEqual({ fresh: "messages" });
  });

  it("[REQ:P0-017b] migrates a version-0 payload to version 1", async () => {
    localStorage.setItem(KEY, JSON.stringify({ version: 0, state: { viewModes: { s1: "messages" } } }));

    await useMessagesViewStore.persist.rehydrate();

    expect(useMessagesViewStore.getState().viewModes).toEqual({ s1: "messages" });
    expect(useMessagesViewStore.getState().positions).toEqual({});
    expect(persisted().version).toBe(1);
  });
});
