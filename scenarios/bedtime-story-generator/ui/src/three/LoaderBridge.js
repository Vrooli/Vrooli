import useExperienceStore from "../state/store.js";

const STEP_INTERVAL = 320;

export default class LoaderBridge {
  constructor() {
    this.store = useExperienceStore;
    this.step = 0;
    this.interval = null;

    const state = this.store.getState();
    if (state.loader.status === "idle") {
      state.startLoader({
        group: "experience-prototype",
        total: 3,
        label: "Initializing immersive prototype",
      });
      this.interval = window.setInterval(() => this._advance(), STEP_INTERVAL);
    }
  }

  _advance() {
    const state = this.store.getState();
    const nextStep = Math.min(this.step + 1, state.loader.total || 3);
    this.step = nextStep;
    state.updateLoader({ loaded: nextStep });

    if (nextStep >= (state.loader.total || 3)) {
      this._complete();
    }
  }

  _complete() {
    if (this.interval) {
      window.clearInterval(this.interval);
      this.interval = null;
    }
    const state = this.store.getState();
    state.completeLoader();
  }

  destroy() {
    if (this.interval) {
      window.clearInterval(this.interval);
      this.interval = null;
    }
    const state = this.store.getState();
    if (state.loader.status !== "idle") {
      state.completeLoader();
    }
  }
}
