import { create } from "zustand";
import { createJSONStorage, persist, type StateStorage } from "zustand/middleware";
import type { PlaybackIntent, PlaybackTarget } from "./types";

interface TtsPlaybackIntentState {
  playbackIntent: PlaybackIntent;
  selectedTarget: PlaybackTarget | null;
  setPlaybackIntent: (intent: PlaybackIntent) => void;
  setSelectedTarget: (target: PlaybackTarget | null) => void;
}

// zustand's JSON storage only reads, writes and removes keys, so the fallback
// implements exactly that. It serves server rendering and browsers that deny
// storage access, keeping playback controls usable for the session.
const memoryStorage = new Map<string, string>();
const memoryStateStorage: StateStorage = {
  getItem: (key) => memoryStorage.get(key) ?? null,
  setItem: (key, value) => { memoryStorage.set(key, value); },
  removeItem: (key) => { memoryStorage.delete(key); },
};

const getSafeStorage = (): StateStorage => {
  if (typeof window === "undefined") return memoryStateStorage;
  try {
    const storage = window.localStorage;
    const probe = "__wc_tts_playback_probe__";
    storage.setItem(probe, "1");
    storage.removeItem(probe);
    return storage;
  } catch {
    return memoryStateStorage;
  }
};

export const useTtsPlaybackIntentStore = create<TtsPlaybackIntentState>()(
  persist(
    (set) => ({
      playbackIntent: "continuous",
      selectedTarget: null,
      setPlaybackIntent: (playbackIntent) => set({ playbackIntent }),
      setSelectedTarget: (selectedTarget) => set({ selectedTarget }),
    }),
    {
      name: "web-console-tts-playback-intent-v1",
      storage: createJSONStorage(getSafeStorage),
      partialize: (state) => ({
        playbackIntent: state.playbackIntent,
        selectedTarget: state.selectedTarget,
      }),
    },
  ),
);
