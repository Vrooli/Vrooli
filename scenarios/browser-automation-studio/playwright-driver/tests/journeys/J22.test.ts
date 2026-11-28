import { createServer } from 'node:http';
import { action, driver, env, openLease, registerLeaseCleanup, registerRecordingCleanup, runAction, withJourneyResources, type Lease } from './support';

function heldInitialPageCallback(): { url: string; ready: Promise<void>; received: Promise<void>; release: () => void; close: () => Promise<void> } {
  let receive!: () => void;
  let release!: () => void;
  const received = new Promise<void>((resolve) => { receive = resolve; });
  const released = new Promise<void>((resolve) => { release = resolve; });
  const server = createServer((request, response) => {
    let body = '';
    request.setEncoding('utf8');
    request.on('data', (chunk: string) => { body += chunk; });
    request.on('end', () => {
      const event = JSON.parse(body) as { eventType?: string };
      if (event.eventType === 'initial') {
        receive();
        void released.then(() => response.writeHead(200).end());
        return;
      }
      response.writeHead(200).end();
    });
  });
  const ready = new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  return {
    ready,
    get url() {
      const address = server.address();
      if (!address || typeof address === 'string') throw new Error('Page callback did not bind a TCP port');
      return `http://127.0.0.1:${address.port}/page-event`;
    },
    received,
    release: () => release(),
    close: () => new Promise<void>((resolve, reject) => {
      server.close((error) => error ? reject(error) : resolve());
    }),
  };
}

describe('[REQ:BAS-RH-J22] replacement capture owns page frames and teardown', () => {
  it('does not let a late pending start displace a newer capture or page', () => withJourneyResources(async (resources) => {
    const lease: Lease = await openLease();
    registerLeaseCleanup(resources, lease);
    const owned = { execution_id: lease.executionId, lease_id: lease.leaseId };
    const callback = heldInitialPageCallback();
    resources.add('release and close page callback server', async () => {
      callback.release();
      await callback.close();
    });
      await callback.ready;
      await runAction(lease, action('navigate', { url: `${env.fixture}/complex` }));
      const record = `/session/${lease.sessionId}/record`;
      const stop = () => driver(`${record}/stop`, owned);

      const pendingStart = driver<{ recording_id: string }>(`${record}/start`, {
        ...owned, recording_id: 'j22-pending', page_callback_url: callback.url,
      });
      registerRecordingCleanup(resources, lease);
      await callback.received;
      await stop();

      // Let the stale generation finish its held callback and reject before
      // starting a replacement. Otherwise its uncommitted event correctly
      // blocks the next recording as an undurable write.
      callback.release();
      await expect(pendingStart).rejects.toThrow('RECORDING_START_SUPERSEDED');
      const { entries } = await driver<{ entries: Array<{ id?: string }> }>(`${record}/actions`);
      const entry_ids = entries.flatMap(({ id }) => id ? [id] : []);
      if (entry_ids.length > 0) await driver(`${record}/actions/ack`, { ...owned, entry_ids });

      const replacement = await driver<{ recording_id: string }>(`${record}/start`, {
        ...owned, recording_id: 'j22-replacement',
      });
      expect(replacement.recording_id).toBe('j22-replacement');
      await expect(driver(`${record}/status`)).resolves.toMatchObject({
        is_recording: true, recording_id: 'j22-replacement',
      });

      const page = await driver<{ driver_page_id: string }>(`${record}/new-page`, {
        ...owned, url: `${env.fixture}/spa/next`,
      });
      expect(page.driver_page_id).not.toBe('');
      const frame = await driver<{ page_url: string; image: string; source: { page_id: string } }>(`${record}/frame`);
      expect(frame.page_url).toBe(`${env.fixture}/spa/next`);
      expect(frame.source.page_id).toBe(page.driver_page_id);
      expect(frame.image).not.toBe('');
      await stop();
  }));
});
