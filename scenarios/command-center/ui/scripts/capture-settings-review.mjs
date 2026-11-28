import { createBASBrowserClientFromEnv } from './bas-browser-client.mjs';
import { captureSettingsReview } from './capture-settings-review-core.mjs';

const outputDir = '/home/matthalloran8/.vrooli/plan-artifacts/command-center-settings-redesign';

try {
  const captures = await captureSettingsReview({
    brokerClient: createBASBrowserClientFromEnv(),
    outputDir,
  });
  console.log(`Captured settings review screenshots at ${captures.map(item => `${item.width}px`).join(' and ')} with no horizontal overflow.`);
} catch (error) {
  console.error(error instanceof Error ? error.message : 'Command Center settings review failed');
  process.exitCode = 1;
}
