import { randomUUID } from 'node:crypto';
import { WebSocketServer, type WebSocket } from 'ws';
import type { StartRecordingRequest, StartRecordingResponse, StreamSettingsRequest, StreamSettingsResponse } from '../../src/routes/record-mode/types';
import { TIMEOUT_MS, action, driver, env, ownership, runAction, withLeases, type Lease } from './support';

type FrameEnvelope = { source: { session_id: string; execution_id: string; lease_id: string; page_id: string }; timing?: { frame_bytes?: number; frame_id?: string } };

/** Resolve with the next value `listen` produces, or reject after the shared wait timeout. */
function next<T>(what: string, listen: (done: (value: T) => void) => () => void): Promise<T> {
  return new Promise((resolve, reject) => {
    const stop = listen((value) => { clearTimeout(timer); stop(); resolve(value); });
    const timer = setTimeout(() => { stop(); reject(new Error(`No ${what} within ${TIMEOUT_MS.wait} ms`)); }, TIMEOUT_MS.wait);
  });
}

const nextSocket = (server: WebSocketServer): Promise<WebSocket> => next('frame stream connection', (done) => {
  server.on('connection', done);
  return () => server.off('connection', done);
});

/** A frame packet is a 4-byte header length, a JSON envelope, then a complete JPEG. */
const nextFrame = (socket: WebSocket): Promise<{ envelope: FrameEnvelope; jpeg: Buffer }> => next('valid frame', (done) => {
  const received = (packet: Buffer): void => {
    if (packet.length < 5) return;
    const headerLength = packet.readUInt32BE(0);
    if (headerLength <= 0 || headerLength > packet.length - 4) return;
    const jpeg = packet.subarray(4 + headerLength);
    if (jpeg.length < 4 || jpeg[0] !== 0xff || jpeg[1] !== 0xd8 || jpeg.at(-2) !== 0xff || jpeg.at(-1) !== 0xd9) return;
    done({ envelope: JSON.parse(packet.subarray(4, 4 + headerLength).toString()) as FrameEnvelope, jpeg: Buffer.from(jpeg) });
  };
  socket.on('message', received);
  return () => socket.off('message', received);
});

async function stopRecording(lease: Lease): Promise<void> {
  const record = `/session/${lease.sessionId}/record`;
  await driver(`${record}/stop`, ownership(lease));
  const { entries } = await driver<{ entries: Array<{ id?: string }> }>(`${record}/actions`);
  const entry_ids = entries.flatMap(({ id }) => (id ? [id] : []));
  if (entry_ids.length > 0) await driver(`${record}/actions/ack`, { ...ownership(lease), entry_ids });
}

describe('[REQ:BAS-RH-J23] frame stream controls and reconnect', () => {
  let observer: WebSocketServer;
  beforeAll(() => new Promise<void>((resolve) => { observer = new WebSocketServer({ host: '127.0.0.1', port: 0 }, resolve); }));
  afterAll(() => {
    for (const socket of observer.clients) socket.terminate();
    return new Promise<void>((resolve) => observer.close(() => resolve()));
  });

  it('given an active frame stream, when controls change and the transport reconnects, then receipts match and a valid frame returns', () => withLeases(async (open) => {
    const lease = await open();
    const record = `/session/${lease.sessionId}/record`;
    expect((await runAction(lease, action('navigate', { url: `${env.fixture}/` }))).finalUrl).toBe(`${env.fixture}/`);

    const address = observer.address();
    if (typeof address === 'string') throw new Error('Frame observer did not bind a TCP port');
    const firstSocket = nextSocket(observer);
    const start: StartRecordingRequest = {
      ...ownership(lease), recording_id: `j23-${randomUUID()}`,
      frame_callback_url: `http://127.0.0.1:${address.port}/frames`, frame_quality: 65, frame_fps: 15,
    };
    expect((await driver<StartRecordingResponse>(`${record}/start`, start)).recording_id).toBe(start.recording_id);
    try {
      const socket = await firstSocket;
      const first = await nextFrame(socket);
      expect(first.envelope.source).toMatchObject({ session_id: lease.sessionId, ...ownership(lease) });

      const controls = { quality: 42, fps: 8, perfMode: true } satisfies StreamSettingsRequest;
      const applied = await driver<StreamSettingsResponse>(`${record}/stream-settings`, controls);
      expect(applied).toMatchObject({ quality: 42, fps: 8, perf_mode: true, is_streaming: true, updated: true });
      expect(Number.isFinite(applied.current_fps)).toBe(true);

      const unsupported = await driver<StreamSettingsResponse>(`${record}/stream-settings`, { scale: 'device' } satisfies StreamSettingsRequest);
      expect(unsupported.scale).toBe('css');
      expect(unsupported.scale_warning).toContain('cannot be changed mid-session');

      const reconnected = nextSocket(observer);
      socket.close(1012, 'exercise frame transport reconnect');
      const afterReconnect = await nextFrame(await reconnected);
      expect(afterReconnect.envelope.source).toEqual(first.envelope.source);
      expect(afterReconnect.envelope.timing).toMatchObject({ frame_bytes: afterReconnect.jpeg.length, frame_id: expect.any(String) });
    } finally {
      await stopRecording(lease);
    }
  }));
});
