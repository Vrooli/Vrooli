import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { PlaybackIntent, PlaybackTarget } from "./types";

interface TtsPlaybackIntentState {
  playbackIntent: PlaybackIntent;
  selectedTarget: PlaybackTarget | null;
  setPlaybackIntent: (intent: PlaybackIntent) => void;
  setSelectedTarget: (target: PlaybackTarget | null) => void;
}

const memoryStorage = new Map<string, string>();

const getSafeStorage = (): Storage => {
  if (typeof window === "undefined") {
    return {
      get length() {
        return memoryStorage.size;
      },
      clear: () => memoryStorage.clear(),
      getItem: (key) => memoryStorage.get(key) ?? null,
      key: (index) => Array.from(memoryStorage.keys())[index] ?? null,
      removeItem: (key) => { memoryStorage.delete(key); },
      setItem: (key, value) => { memoryStorage.set(key, value); },
    };
  }
  try {
    const storage = window.localStorage;
    const probe = "__wc_tts_playback_probe__";
    storage.setItem(probe, "1");
    storage.removeItem(probe);
    return storage;
  } catch {
    return {
      get length() {
        return memoryStorage.size;
      },
      clear: () => memoryStorage.clear(),
      getItem: (key) => memoryStorage.get(key) ?? null,
      key: (index) => Array.from(memoryStorage.keys())[index] ?? null,
      removeItem: (key) => { memoryStorage.delete(key); },
      setItem: (key, value) => { memoryStorage.set(key, value); },
    };
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
