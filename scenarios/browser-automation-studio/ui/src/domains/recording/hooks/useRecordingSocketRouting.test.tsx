import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { WebSocketProvider } from '../../../contexts/WebSocketProvider';
import { useWebSocket } from '../../../contexts/WebSocketContext';

class FakeSocket {
  static OPEN = 1;
  static CONNECTING = 0;
  static instances: FakeSocket[] = [];

  readyState = FakeSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent<unknown>) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  sent: string[] = [];

  constructor(readonly url: string) { FakeSocket.instances.push(this); }
  send(data: string) { this.sent.push(data); }
  open() { this.readyState = FakeSocket.OPEN; this.onopen?.(new Event('open')); }
  close(code = 1000, reason = '') {
    this.readyState = 3;
    this.onclose?.({code, reason} as CloseEvent);
  }
}

describe('WebSocketProvider session stream routing', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    FakeSocket.instances = [];
  });

  it('routes recording subscriptions and input through the exact session socket', () => {
    vi.stubGlobal('WebSocket', FakeSocket);
    const wrapper = ({children}: {children: React.ReactNode}) => <WebSocketProvider>{children}</WebSocketProvider>;
    const {result, unmount} = renderHook(() => useWebSocket(), {wrapper});
    const sessionID = 'b6a0a70d-503b-4ff2-8a9b-f95627f2f45d';

    act(() => {
      result.current.send({type: 'subscribe_recording', session_id: sessionID, frames: false});
    });
    expect(FakeSocket.instances).toHaveLength(2);
    const sessionSocket = FakeSocket.instances[1];
    expect(sessionSocket.url).toBe(`ws://${window.location.host}/ws/sessions/${sessionID}`);

    act(() => sessionSocket.open());
    expect(sessionSocket.sent).toEqual([JSON.stringify({type: 'subscribe_recording', session_id: sessionID, frames: false})]);

    act(() => {
      result.current.send({type: 'recording_input', session_id: sessionID, input: {type: 'click'}});
    });
    expect(sessionSocket.sent[1]).toBe(JSON.stringify({type: 'recording_input', session_id: sessionID, input: {type: 'click'}}));
    expect(FakeSocket.instances[0].sent).toHaveLength(0);

    unmount();
  });
});
