import { spawn, execFile } from 'node:child_process';
import { once } from 'node:events';
import { readFileSync } from 'node:fs';
import type { Page } from 'rebrowser-playwright';

export interface PipeWireDeviceEvidence {
  strategy: 'host_device';
  deviceName: string;
  sourceNodeName: string;
  sinkNodeName: string;
  sourceVisible: boolean;
  sinkVisible: boolean;
}

export interface PipeWireQualificationDevice {
  evidence: PipeWireDeviceEvidence;
  startWavLoop(
    path: string,
    pauseMs?: number,
    onFailure?: (error: Error) => void
  ): () => Promise<void>;
  close(): Promise<void>;
}

export interface BrowserCaptureDeviceEvidence {
  enumerated: boolean;
  selectedLabel: string;
  sampleRate: number;
  channelCount: number;
}

function isNavigationContextError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error);
  return /execution context was destroyed|most likely because of a navigation|navigation interrupted|target page, context or browser has been closed/i.test(message);
}

const SOURCE_NODE = 'Vrooli_Qualification_Microphone';
const SINK_NODE = 'Vrooli_Qualification_Sink';
const SINK_NODE_NAME = 'vrooli_qualification_sink';
export const PIPEWIRE_QUALIFICATION_DEVICE_NAME = 'Vrooli Qualification Microphone';

function wavPcmPayload(path: string): Buffer {
  const wav = readFileSync(path);
  if (
    wav.length < 12 ||
    wav.toString('ascii', 0, 4) !== 'RIFF' ||
    wav.toString('ascii', 8, 12) !== 'WAVE'
  ) {
    throw new Error(`qualification playback requires a RIFF/WAVE file: ${path}`);
  }
  let offset = 12;
  let format: Buffer | undefined;
  let data: Buffer | undefined;
  while (offset + 8 <= wav.length) {
    const chunkID = wav.toString('ascii', offset, offset + 4);
    const chunkSize = wav.readUInt32LE(offset + 4);
    offset += 8;
    if (offset + chunkSize > wav.length) throw new Error(`invalid WAV chunk in ${path}`);
    if (chunkID === 'fmt ') format = wav.subarray(offset, offset + chunkSize);
    if (chunkID === 'data') {
      data = wav.subarray(offset, offset + chunkSize);
      break;
    }
    offset += chunkSize + (chunkSize % 2);
  }
  if (!format || format.length < 16 || !data) throw new Error(`WAV has no PCM data chunk: ${path}`);
  if (
    format.readUInt16LE(0) !== 1 ||
    format.readUInt16LE(2) !== 1 ||
    format.readUInt32LE(4) !== 16000 ||
    format.readUInt16LE(14) !== 16
  ) {
    throw new Error(`qualification playback requires 16 kHz mono PCM16 WAV: ${path}`);
  }
  return data;
}

function commandAvailable(command: string): Promise<boolean> {
  return new Promise((resolve) => {
    execFile('which', [command], (error) => resolve(!error));
  });
}

function statusText(): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile('wpctl', ['status'], { encoding: 'utf8' }, (error, stdout, stderr) => {
      if (error) reject(new Error(`wpctl status failed: ${stderr || error.message}`));
      else resolve(stdout);
    });
  });
}

function runWpctl(args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile('wpctl', args, { encoding: 'utf8' }, (error, stdout, stderr) => {
      if (error) reject(new Error(`wpctl ${args[0]} failed: ${stderr || error.message}`));
      else resolve(stdout);
    });
  });
}

async function waitForStatus(source: string, sink: string): Promise<string> {
  const deadline = Date.now() + 2_000;
  let last = '';
  while (Date.now() < deadline) {
    last = await statusText();
    if (last.includes(source) && last.includes(sink)) return last;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`PipeWire qualification nodes did not appear: ${source}, ${sink}`);
}

/** Proves that the browser enumerated and opened the host-created input. */
export async function verifyBrowserCaptureDevice(
  page: Page,
  expectedLabel: string
): Promise<BrowserCaptureDeviceEvidence> {
  // The qualification origin can perform a client-side redirect immediately
  // after DOMContentLoaded. If that happens between enumerateDevices and the
  // page evaluation boundary, Playwright destroys the execution context. This
  // is a navigation race, not evidence that the host device disappeared; retry
  // the same read against the new document before failing setup.
  for (let attempt = 0; attempt < 3; attempt += 1) {
    try {
      await page.waitForLoadState('domcontentloaded', { timeout: 2_000 }).catch(() => undefined);
      return await page.evaluate(async (label) => {
        const devices = await navigator.mediaDevices.enumerateDevices();
        const input = devices.find(
          (device) => device.kind === 'audioinput' && device.label.includes(label)
        );
        if (!input) return { enumerated: false, selectedLabel: '', sampleRate: 0, channelCount: 0 };
        const stream = await navigator.mediaDevices.getUserMedia({
          audio: { deviceId: { exact: input.deviceId } },
        });
        try {
          const track = stream.getAudioTracks()[0];
          const settings = track?.getSettings();
          return {
            enumerated: true,
            selectedLabel: track?.label ?? input.label,
            sampleRate: settings?.sampleRate ?? 0,
            channelCount: settings?.channelCount ?? 0,
          };
        } finally {
          for (const track of stream.getTracks()) track.stop();
        }
      }, expectedLabel);
    } catch (error) {
      if (!isNavigationContextError(error) || attempt === 2) throw error;
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
  }
  throw new Error('browser capture device verification did not complete');
}

/**
 * Creates a user-owned PipeWire source/sink pair for automated device
 * qualification. This is intentionally Linux/PipeWire-specific; callers must
 * record not_measured on other hosts instead of silently substituting a fake
 * browser source. No sudo, system mutation, or persistent host repair is used.
 */
export async function createPipeWireQualificationDevice(): Promise<PipeWireQualificationDevice> {
  if (process.platform !== 'linux')
    throw new Error('PipeWire device evidence is only available on Linux');
  if (
    !(await commandAvailable('pw-loopback')) ||
    !(await commandAvailable('pw-cat')) ||
    !(await commandAvailable('wpctl'))
  ) {
    throw new Error('pw-loopback, pw-cat, and wpctl are required for device evidence');
  }

  const loopback = spawn(
    'pw-loopback',
    [
      '--name',
      'vrooli-qualification-device',
      '--channels',
      '1',
      '--latency',
      '100',
      // pw-loopback's capture side is the injected playback endpoint and its
      // playback side is the browser-visible capture endpoint. Reversing these
      // media classes is what makes pw-cat audio arrive at getUserMedia.
      '--capture-props',
      `media.class=Audio/Sink node.name=vrooli_qualification_sink node.description=${SINK_NODE}`,
      '--playback-props',
      `media.class=Audio/Source node.name=vrooli_qualification_microphone node.description=${PIPEWIRE_QUALIFICATION_DEVICE_NAME.replaceAll(' ', '_')}`,
    ],
    { stdio: ['ignore', 'pipe', 'pipe'] }
  );
  let stderr = '';
  loopback.stderr?.on('data', (chunk: Buffer) => {
    stderr += chunk.toString();
  });

  try {
    const status = await waitForStatus(SOURCE_NODE, SINK_NODE);
    const sourceID = status.match(new RegExp(`\\b(\\d+)\\.\\s+${SOURCE_NODE}\\s`))?.[1];
    if (!sourceID)
      throw new Error(`PipeWire qualification source id was not visible: ${SOURCE_NODE}`);
    let previousDefaultSourceID: string | undefined;
    try {
      const previous = await runWpctl(['inspect', '@DEFAULT_SOURCE@']);
      if (previous.includes('media.class = "Audio/Source"')) {
        previousDefaultSourceID = previous.match(/^id (\\d+),/m)?.[1];
      }
    } catch {
      // No default capture node is a valid host state; clear our temporary
      // default on close so WirePlumber can select one again.
    }
    await runWpctl(['set-default', sourceID]);
    let closed = false;
    const restoreDefaultSource = async (): Promise<void> => {
      if (previousDefaultSourceID)
        await runWpctl(['set-default', previousDefaultSourceID]).catch(() => undefined);
      else await runWpctl(['clear-default', '1']).catch(() => undefined);
    };
    return {
      evidence: {
        strategy: 'host_device',
        deviceName: PIPEWIRE_QUALIFICATION_DEVICE_NAME,
        sourceNodeName: SOURCE_NODE,
        sinkNodeName: SINK_NODE,
        sourceVisible: status.includes(SOURCE_NODE),
        sinkVisible: status.includes(SINK_NODE),
      },
      startWavLoop: (path: string, pauseMs = 0, onFailure?: (error: Error) => void) => {
        const pcm = wavPcmPayload(path);
        const silence =
          pauseMs > 0 ? Buffer.alloc(Math.round((16000 * pauseMs) / 1000) * 2) : undefined;
        let stopped = false;
        let stopResolve: (() => void) | undefined;
        const stopSignal = new Promise<void>((resolve) => {
          stopResolve = resolve;
        });
        const player = spawn(
          'pw-cat',
          [
            '--playback',
            '--target',
            SINK_NODE_NAME,
            '--rate',
            '16000',
            '--channels',
            '1',
            '--format',
            's16',
            '-',
          ],
          { stdio: ['pipe', 'ignore', 'pipe'] }
        );
        let playerError = '';
        let playbackFailure: Error | undefined;
        player.stderr?.on('data', (chunk: Buffer) => {
          playerError += chunk.toString();
        });
        const exitPromise = new Promise<void>((resolve, reject) => {
          player.once('error', reject);
          player.once('exit', (code) => {
            if (code === 0 || stopped) resolve();
            else reject(new Error(`pw-cat exited ${code}: ${playerError}`));
          });
        });
        const write = async (payload: Buffer): Promise<void> => {
          if (stopped || !player.stdin) return;
          if (player.stdin.write(payload)) return;
          await Promise.race([once(player.stdin, 'drain').then(() => undefined), stopSignal]);
        };
        const loopPromise = (async () => {
          while (!stopped) {
            await write(pcm);
            if (silence) await write(silence);
          }
          player.stdin?.end();
          await exitPromise;
        })();
        void loopPromise.catch((error: unknown) => {
          if (stopped) return;
          playbackFailure =
            error instanceof Error
              ? error
              : new Error(`qualification playback failed: ${String(error)}`);
          onFailure?.(playbackFailure);
          stopped = true;
          stopResolve?.();
          player.stdin?.destroy();
          player.kill('SIGTERM');
        });

        return async () => {
          if (!stopped) {
            stopped = true;
            stopResolve?.();
            player.stdin?.destroy();
            player.kill('SIGTERM');
          }
          await Promise.race([
            loopPromise.catch(() => undefined),
            exitPromise.catch(() => undefined),
          ]);
          if (playbackFailure) throw playbackFailure;
        };
      },
      close: async () => {
        if (closed) return;
        closed = true;
        if (loopback.exitCode === null) {
          loopback.kill('SIGTERM');
          await Promise.race([
            once(loopback, 'exit'),
            new Promise((resolve) => setTimeout(resolve, 2_000)),
          ]);
          if (loopback.exitCode === null) loopback.kill('SIGKILL');
        }
        await restoreDefaultSource();
      },
    };
  } catch (error) {
    loopback.kill('SIGTERM');
    await once(loopback, 'exit').catch(() => undefined);
    throw new Error(
      `${error instanceof Error ? error.message : String(error)}${stderr ? ` (${stderr.trim()})` : ''}`
    );
  }
}
