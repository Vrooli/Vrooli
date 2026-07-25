import { chromium } from 'rebrowser-playwright';
import type { BrowserContext } from 'rebrowser-playwright';
import { ServiceWorkerController } from './service-worker';
import { createRecordingContextInitializer } from './recording';
import { logger } from './utils';

const WAV = '/home/matthalloran8/Vrooli/scenarios/audio-tools/bas/fixtures/dictation-reference.wav';
const ARGS = [
  '--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream',
  `--use-file-for-fake-audio-capture=${WAV}`,
  '--headless=new', '--no-sandbox', '--disable-dev-shm-usage', '--ozone-platform=headless',
];

const PROBE = async () => {
  const r: any = {};
  const ac = new AudioContext();
  try { await ac.resume(); } catch (e) { r.rErr = String(e); }
  r.state = ac.state;
  const t0 = ac.currentTime;
  const sp = ac.createScriptProcessor(4096, 1, 1);
  r.cb = 0; r.micMax = 0;
  sp.onaudioprocess = (e) => {
    r.cb++;
    const d = e.inputBuffer.getChannelData(0);
    for (let i = 0; i < d.length; i++) { const a = Math.abs(d[i]); if (a > r.micMax) r.micMax = a; }
  };
  try {
    const s = await navigator.mediaDevices.getUserMedia({ audio: true });
    ac.createMediaStreamSource(s).connect(sp);
  } catch (e) { r.gum = String(e).slice(0, 50); }
  sp.connect(ac.destination);
  await new Promise((x) => setTimeout(x, 2500));
  r.clock = +(ac.currentTime - t0).toFixed(2);
  return r;
};

async function run(name: string, apply: (ctx: BrowserContext) => Promise<void>, opts: any = {}) {
  let out: any;
  const b = await chromium.launch({ headless: false, args: ARGS });
  try {
    const ctx = await b.newContext({ viewport: { width: 1280, height: 720 }, ...opts });
    await apply(ctx);
    const p = await ctx.newPage();
    await p.goto('http://localhost:20004/', { waitUntil: 'load', timeout: 20000 });
    out = await p.evaluate(PROBE);
  } catch (e) { out = { fatal: String(e).slice(0, 160) }; }
  await b.close();
  const broken = !out.cb;
  console.log(`${broken ? 'BROKEN ' : 'ok     '} ${name.padEnd(34)} ${JSON.stringify(out)}`);
}

(async () => {
  await run('A control', async () => {});
  await run('B service-worker controller', async (ctx) => {
    const swc = new ServiceWorkerController('bisect', { mode: 'allow' });
    await swc.setupBlockingForContext(ctx);
  });
  await run('C recording initializer', async (ctx) => {
    const init = createRecordingContextInitializer({ logger });
    await (init as any).initialize(ctx);
  });
  await run('D second context in browser', async () => {});
  process.exit(0);
})();
