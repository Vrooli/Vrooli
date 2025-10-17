import useExperienceStore from "../state/store.js";

const PRESET_HOTSPOTS = {
  studio: [
    { id: "wide", label: "Cinematic Wide" },
    { id: "bookshelf", label: "Bookshelf" },
    { id: "bed", label: "Bed & Reader" },
    { id: "window", label: "Window" },
    { id: "mobile", label: "Ceiling Mobile" },
  ],
  bedroom: [
    { id: "wide", label: "Dreamy Wide" },
    { id: "bookshelf", label: "Story Shelves" },
    { id: "bed", label: "Bedtime Nook" },
    { id: "window", label: "Stargazing Window" },
    { id: "mobile", label: "Ceiling Mobile" },
  ],
};

export default class HotspotRegistry {
  constructor() {
    this.store = useExperienceStore;
    Object.entries(PRESET_HOTSPOTS).forEach(([roomId, entries]) => {
      this.store.getState().registerHotspots(entries, { roomId });
    });
  }

  destroy() {}
}
