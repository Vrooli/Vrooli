import EventEmitter from "./EventEmitter.js";

const MAX_PIXEL_RATIO = 1.0;

export default class Sizes extends EventEmitter {
  constructor({ target }) {
    super();

    this.target = target;
    this.pixelRatio = Math.min(
      Math.max(window.devicePixelRatio || 1, 1),
      MAX_PIXEL_RATIO,
    );
    this._onResize = this._onResize.bind(this);

    this._measure();
    window.addEventListener("resize", this._onResize);
  }

  _measure() {
    const bounds = this.target?.getBoundingClientRect?.();

    this.width = bounds?.width || window.innerWidth;
    this.height = bounds?.height || window.innerHeight;
    this.pixelRatio = Math.min(
      Math.max(window.devicePixelRatio || 1, 1),
      MAX_PIXEL_RATIO,
    );
  }

  _onResize() {
    this._measure();
    this.emit("resize", {
      width: this.width,
      height: this.height,
      pixelRatio: this.pixelRatio,
    });
  }

  destroy() {
    window.removeEventListener("resize", this._onResize);
  }
}
