export default class EventEmitter {
  constructor() {
    this.listeners = {};
  }

  on(event, handler) {
    if (!event || typeof handler !== "function") {
      return () => {};
    }

    if (!this.listeners[event]) {
      this.listeners[event] = new Set();
    }

    this.listeners[event].add(handler);

    return () => this.off(event, handler);
  }

  off(event, handler) {
    const handlers = this.listeners[event];
    if (!handlers) {
      return;
    }

    handlers.delete(handler);

    if (handlers.size === 0) {
      delete this.listeners[event];
    }
  }

  emit(event, ...args) {
    const handlers = this.listeners[event];
    if (!handlers) {
      return;
    }

    handlers.forEach((handler) => {
      try {
        handler(...args);
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error(`[EventEmitter] handler for "${event}" failed`, error);
      }
    });
  }

  removeAllListeners() {
    this.listeners = {};
  }
}
