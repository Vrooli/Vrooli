import { describe, expect, it, vi } from "vitest";
import { matchesSavedDraft, removeSavedDrafts, saveCatalogBatch, type CatalogChange } from "./catalogSaveBatch";

const changes: CatalogChange[] = [
  { catalog: "themes", entry: { id: "cosmos", color: "blue" } },
  { catalog: "rooms", entry: { id: "mission-control", theme: "cosmos" } },
  { catalog: "signals", entry: { id: "active-scenarios", source: "local" } },
];
const [themeChange, roomChange, signalChange] = changes as [CatalogChange, CatalogChange, CatalogChange];

describe("catalog save batches", () => {
  it("stops at the first failure and reports the successful prefix", async () => {
    const save = vi.fn((_catalog: string, entry: { id: string }): Promise<void> => {
      if (entry.id === "mission-control") return Promise.reject(new Error("offline"));
      return Promise.resolve();
    });

    const result = await saveCatalogBatch(changes, save);

    expect(save.mock.calls.map(([catalog]) => catalog)).toEqual(["themes", "rooms"]);
    expect(result).toEqual({ saved: [themeChange], failed: { change: roomChange, error: new Error("offline") } });
  });

  it("removes only unchanged successful snapshots, keeping edits made in flight", () => {
    const saved = [themeChange, roomChange];
    const drafts = {
      "themes:cosmos": themeChange.entry,
      "rooms:mission-control": { ...roomChange.entry, theme: "new-theme" },
      "signals:active-scenarios": signalChange.entry,
    };

    expect(removeSavedDrafts(drafts, saved)).toEqual({
      "rooms:mission-control": { id: "mission-control", theme: "new-theme" },
      "signals:active-scenarios": signalChange.entry,
    });
    expect(matchesSavedDraft({ catalog: "rooms", entry: roomChange.entry }, saved)).toBe(true);
    expect(matchesSavedDraft({ catalog: "rooms", entry: drafts["rooms:mission-control"] }, saved)).toBe(false);
  });

  it("returns an empty successful batch without calling the writer", async () => {
    const save = vi.fn();
    expect(await saveCatalogBatch([], save)).toEqual({ saved: [], failed: null });
    expect(save).not.toHaveBeenCalled();
  });
});
