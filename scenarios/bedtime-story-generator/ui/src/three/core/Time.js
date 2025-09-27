import EventEmitter from "./EventEmitter.js";

export default class Time extends EventEmitter {
  constructor() {
    super();

    this.start = performance.now();
    this.current = this.start;
    this.elapsed = 0;
    this.delta = 16;
    this._active = true;

    this._tick = this._tick.bind(this);
    this._raf = requestAnimationFrame(this._tick);
  }

  _tick(now) {
    if (!this._active) {
      return;
    }

    this.delta = now - this.current;
    this.elapsed = now - this.start;
    this.current = now;

    // Cap extremely large delta (tab switching etc.)
    if (this.delta > 100) {
      this.delta = 100;
    }

    this.emit("tick", {
      delta: this.delta,
      elapsed: this.elapsed,
      now,
    });

    this._raf = requestAnimationFrame(this._tick);
  }

  stop() {
    this._active = false;
    if (this._raf) {
      cancelAnimationFrame(this._raf);
      this._raf = null;
    }
  }
}
