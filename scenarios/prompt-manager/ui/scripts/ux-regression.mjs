import { createBASUXClientFromEnv } from './bas-ux-client.mjs';
import { runUXRegression } from './ux-regression-bas-core.mjs';

try {
  const result = await runUXRegression({ brokerClient: createBASUXClientFromEnv() });
  console.log(`Prompt Manager UX regression passed through BAS: ${result.checks.join(', ')}.`);
} catch (error) {
  console.error(error instanceof Error ? error.message : 'Prompt Manager UX regression failed');
  process.exitCode = 1;
}
