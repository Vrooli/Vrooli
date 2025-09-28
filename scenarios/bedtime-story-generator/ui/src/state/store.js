import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";
import { getTimeOfDay, getThemeForTime } from "../utils/timeUtils";

const initialTimeOfDay = getTimeOfDay();
const initialTheme = getThemeForTime();

const ROOM_PRESETS = {
  studio: {
    label: "Creator Studio",
    defaultFocus: "wide",
    hotspots: {
      wide: { id: "wide", label: "Cinematic Wide" },
      bookshelf: { id: "bookshelf", label: "Bookshelf" },
      bed: { id: "bed", label: "Bed & Reader" },
      window: { id: "window", label: "Window" },
      mobile: { id: "mobile", label: "Ceiling Mobile" },
      lamp: { id: "lamp", label: "Reading Lamp" },
      toyChest: { id: "toyChest", label: "Toy Chest" },
    },
  },
  bedroom: {
    label: "Dream Suite",
    defaultFocus: "wide",
    hotspots: {
      wide: { id: "wide", label: "Dreamy Wide" },
      bookshelf: { id: "bookshelf", label: "Story Shelves" },
      bed: { id: "bed", label: "Bedtime Nook" },
      window: { id: "window", label: "Stargazing Window" },
      mobile: { id: "mobile", label: "Ceiling Mobile" },
      lamp: { id: "lamp", label: "Reading Lamp" },
      toyChest: { id: "toyChest", label: "Toy Chest" },
    },
  },
};

const cloneRooms = () =>
  Object.fromEntries(
    Object.entries(ROOM_PRESETS).map(([id, room]) => [
      id,
      {
        label: room.label,
        defaultFocus: room.defaultFocus,
        hotspots: Object.fromEntries(
          Object.entries(room.hotspots).map(([hotspotId, hotspot]) => [
            hotspotId,
            { ...hotspot },
          ]),
        ),
      },
    ]),
  );

const buildHotspotState = (room, requestedFocus) => {
  const hotspotKeys = Object.keys(room.hotspots);
  const fallbackFocus =
    room.defaultFocus || hotspotKeys[0] || "wide";
  const focus = room.hotspots[requestedFocus] ? requestedFocus : fallbackFocus;

  const map = Object.fromEntries(
    Object.entries(room.hotspots).map(([id, hotspot]) => [
      id,
      { ...hotspot, active: id === focus },
    ]),
  );

  return { map, focus };
};

const roomsInitial = cloneRooms();
const initialRoomId = "bedroom";
const initialHotspotState = buildHotspotState(
  roomsInitial[initialRoomId],
  roomsInitial[initialRoomId].defaultFocus,
);

const defaultLoaderState = {
  status: "idle",
  group: null,
  label: null,
  loaded: 0,
  total: 0,
  progress: 0,
};

const useExperienceStore = create(
  subscribeWithSelector((set, get) => ({
    timeOfDay: initialTimeOfDay,
    theme: initialTheme,
    stories: [],
    selectedStory: null,
    isReading: false,
    generatorOpen: false,
    parentDashboardOpen: false,
    prototypeOpen: false,
    developerMode: false,
    developerConsoleOpen: false,
    rooms: roomsInitial,
    availableRooms: Object.entries(roomsInitial).map(([id, room]) => ({
      id,
      label: room.label,
    })),
    activeRoom: initialRoomId,
    cameraFocus: initialHotspotState.focus,
    cameraAutopilot: true,
    hotspots: initialHotspotState.map,
    loader: defaultLoaderState,

    setTimeOfDay: (timeOfDay) => {
      set({ timeOfDay, theme: getThemeForTime(timeOfDay) });
    },

    setStories: (stories) => set({ stories }),

    prependStory: (story) =>
      set((state) => ({ stories: [story, ...state.stories] })),

    openStory: (story) => set({ selectedStory: story, isReading: true }),

    closeStory: () => set({ isReading: false }),

    clearSelectedStory: () => set({ selectedStory: null }),

    setGeneratorOpen: (open) => set({ generatorOpen: open }),

    setParentDashboardOpen: (open) => set({ parentDashboardOpen: open }),

    setPrototypeOpen: (open) => set({ prototypeOpen: open }),

    setDeveloperMode: (open) => set({ developerMode: open }),

    setDeveloperConsoleOpen: (open) => set({ developerConsoleOpen: open }),

    setCameraFocus: (id) => {
      const state = get();
      const room = state.rooms[state.activeRoom];
      if (!room?.hotspots[id]) {
        return;
      }

      const { map, focus } = buildHotspotState(room, id);

      set({ cameraFocus: focus, hotspots: map });
    },

    setCameraAutopilot: (enabled) => {
      set({ cameraAutopilot: enabled });
    },

    setActiveRoom: (roomId) => {
      const state = get();
      const room = state.rooms[roomId];
      if (!room) return;

      const { map, focus } = buildHotspotState(room, room.defaultFocus);
      set({ activeRoom: roomId, hotspots: map, cameraFocus: focus });
    },

    registerHotspots: (entries = [], options = {}) => {
      if (!entries.length) return;

      const { roomId } = options;
      const state = get();
      const targetRoomId = roomId || state.activeRoom;
      const rooms = { ...state.rooms };
      const room = rooms[targetRoomId];
      if (!room) return;

      room.hotspots = { ...room.hotspots };

      entries.forEach(({ id, label }) => {
        if (!id) return;
        const current = room.hotspots[id];
        room.hotspots[id] = {
          id,
          label: label || current?.label || id,
        };
      });

      rooms[targetRoomId] = room;

      const updates = { rooms };
      if (targetRoomId === state.activeRoom) {
        const { map, focus } = buildHotspotState(room, state.cameraFocus);
        updates.hotspots = map;
        updates.cameraFocus = focus;
      }

      set(updates);
    },

    setHotspotActive: (id, active) => {
      const prev = get().hotspots;
      if (!prev[id]) return;
      set({
        hotspots: {
          ...prev,
          [id]: { ...prev[id], active },
        },
      });
    },

    startLoader: ({ group, total = 0, label = null } = {}) => {
      set({
        loader: {
          status: "loading",
          group: group || null,
          label,
          total,
          loaded: 0,
          progress: total > 0 ? 0 : null,
        },
      });
    },

    updateLoader: ({ loaded, total, label, group }) => {
      const current = get().loader;
      const nextTotal = typeof total === "number" ? total : current.total;
      const nextLoaded = typeof loaded === "number" ? loaded : current.loaded;
      const progress =
        nextTotal > 0 ? Math.min(nextLoaded / nextTotal, 1) : null;

      set({
        loader: {
          status: "loading",
          group: group ?? current.group,
          label: label ?? current.label,
          total: nextTotal,
          loaded: nextLoaded,
          progress,
        },
      });
    },

    completeLoader: () => {
      const current = get().loader;
      set({
        loader: {
          ...defaultLoaderState,
          status: "complete",
          progress: 1,
          loaded: current.total,
        },
      });
    },

    resetLoader: () => set({ loader: defaultLoaderState }),
  })),
);

export default useExperienceStore;
export const getExperienceState = () => useExperienceStore.getState();
export const subscribeExperienceStore = (selector, listener) =>
  useExperienceStore.subscribe(selector, listener);
