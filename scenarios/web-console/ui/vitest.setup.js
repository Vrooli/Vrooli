if (typeof globalThis.WebSocket === 'undefined') {
  class MockWebSocket {
    static CONNECTING = 0
    static OPEN = 1
    static CLOSING = 2
    static CLOSED = 3

    constructor() {
      this.readyState = MockWebSocket.OPEN
    }

    send() {}
    close() {
      this.readyState = MockWebSocket.CLOSED
    }
    addEventListener() {}
    removeEventListener() {}
  }

  globalThis.WebSocket = MockWebSocket
}
