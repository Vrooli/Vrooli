import { randomUUID } from 'node:crypto';
import { WebSocketServer, type WebSocket } from 'ws';
import type { StartRecordingRequest, StartRecordingResponse } from '../../src/routes/record-mode/types';
import { TIMEOUT_MS, action, driver, env, fixtureInput, ownership, registerRecordingCleanup, runAction, withLeases, type Lease } from './support';

type FrameSource = { stream_kind: 'recording'; session_id: string; execution_id: string; lease_id: string; page_id: string };
type FrameEnvelope = { source: FrameSource };

function next<T>(what: string, listen: (done: (value: T) => void) => () => void): Promise<T> {
  return new Promise((resolve, reject) => {
    const stop = listen((value) => { clearTimeout(timer); stop(); resolve(value); });
    const timer = setTimeout(() => { stop(); reject(new Error(`No ${what} within ${TIMEOUT_MS.wait} ms`)); }, TIMEOUT_MS.wait);
  });
}

function nextSocket(server: WebSocketServer): Promise<WebSocket> {
  return next('frame stream connection', (done) => {
    server.on('connection', done);
    return () => server.off('connection', done);
  });
}

function nextFrame(socket: WebSocket): Promise<FrameEnvelope> {
  return next('valid frame', (done) => {
    const received = (packet: Buffer): void => {
      if (packet.length < 5) return;
      const headerLength = packet.readUInt32BE(0);
      if (headerLength <= 0 || headerLength > packet.length - 4) return;
      const jpeg = packet.subarray(4 + headerLength);
      if (jpeg.length < 4 || jpeg[0] !== 0xff || jpeg[1] !== 0xd8 || jpeg.at(-2) !== 0xff || jpeg.at(-1) !== 0xd9) return;
      done({ source: (JSON.parse(packet.subarray(4, 4 + headerLength).toString()) as FrameEnvelope).source });
    };
    socket.on('message', received);
    return () => socket.off('message', received);
  });
}

describe('[REQ:BAS-RH-J05] bounded frame viewer recovery', () => {
  let observer: WebSocketServer;

  beforeAll(() => new Promise<void>((resolve) => {
    observer = new WebSocketServer({ host: '127.0.0.1', port: 0 }, resolve);
  }));

  afterAll(() => {
    for (const socket of observer.clients) socket.terminate();
    return new Promise<void>((resolve) => observer.close(() => resolve()));
  });

  it('reconnects a slow viewer and only delivers frames for the selected page', () => withLeases(async (open, resources) => {
    const lease: Lease = await open();
    expect((await runAction(lease, action('navigate', { url: `${env.fixture}/` }))).finalUrl).toBe(`${env.fixture}/`);
    const address = observer.address();
    if (typeof address === 'string') throw new Error('Frame observer did not bind a TCP port');

    const firstConnection = nextSocket(observer);
    const record = `/session/${lease.sessionId}/record`;
    const start: StartRecordingRequest = {
      ...ownership(lease), recording_id: `j05-${randomUUID()}`,
      frame_stream_url: `ws://127.0.0.1:${address.port}/frames`, frame_quality: 65, frame_fps: 15,
    };
    expect((await driver<StartRecordingResponse>(`${record}/start`, start)).recording_id).toBe(start.recording_id);
    registerRecordingCleanup(resources, lease);
    const slowViewer = await firstConnection;
    const initial = await nextFrame(slowViewer);
    expect(initial.source).toMatchObject({ stream_kind: 'recording', session_id: lease.sessionId, ...ownership(lease) });

    // Stop reading while the selected tab changes, then disconnect. The new
    // connection must start with a valid frame owned by the replacement page.
    slowViewer.pause();
    const page = await driver<{ driver_page_id: string }>(`${record}/new-page`, {
      ...ownership(lease), url: `${env.fixture}/gestures`,
    });
    expect(page.driver_page_id).not.toBe(initial.source.page_id);
    const typed = `viewer-paused-${randomUUID()}`;
    await runAction(lease, action('input', { selector: '#shortcut', value: typed }));
    expect((await fixtureInput((input) => input.value === typed)).value).toBe(typed);

    const recoveredConnection = nextSocket(observer);
    // Deliver the close handshake after the pause window so the driver can
    // observe the disconnect and run its reconnect path.
    slowViewer.resume();
    slowViewer.close(1012, 'viewer detached');
    const recovered = await nextFrame(await recoveredConnection);
    expect(recovered.source).toMatchObject({
      stream_kind: 'recording', session_id: lease.sessionId, ...ownership(lease), page_id: page.driver_page_id,
    });
  }));
});
