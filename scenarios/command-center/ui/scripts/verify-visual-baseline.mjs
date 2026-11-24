import pngjs from "/home/matthalloran8/Vrooli/scenarios/prompt-manager/ui/node_modules/pngjs/lib/png.js";
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createBASBrowserClientFromEnv } from './bas-browser-client.mjs';
import { verifyVisualBaseline } from './verify-visual-baseline-core.mjs';

const scriptRoot = fileURLToPath(new URL('../', import.meta.url));
const configRoot = fileURLToPath(new URL('../../config/rooms/', import.meta.url));
const origin = process.env.COMMAND_CENTER_UI_URL || `http://127.0.0.1:${process.env.UI_PORT || '21235'}`;
const accept = process.env.VISUAL_BASELINE_ACCEPT === '1';
const { PNG } = pngjs;
const message = await verifyVisualBaseline({
  brokerClient: createBASBrowserClientFromEnv(),
  baseURL: origin,
  configRoot,
  baselineRoot: join(scriptRoot, 'baseline'),
  accept,
  png: PNG,
});
console.log(message);
