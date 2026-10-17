import { timelineEntryToJson, type TimelineEntry } from '../../proto/recording';
import { metrics } from '../../utils';

/** A matching receipt is issued by the Go ingress only after journal commit. */
export async function streamRecordingEntry(
  callbackUrl: string,
  entry: TimelineEntry,
  routedTestMode = false
): Promise<void> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);
  let failureReason = 'network';
  try {
    const response = await fetch(callbackUrl, {
      method: 'POST', headers: {
        'Content-Type': 'application/json',
        ...(routedTestMode ? { 'X-Vrooli-Test-Mode': '1' } : {}),
      },
      body: JSON.stringify(timelineEntryToJson(entry)), signal: controller.signal,
    });
    failureReason = 'http_error';
    if (!response.ok) throw new Error(`Recording callback returned ${response.status}`);
    const receipt = await response.json() as { status?: string; entry_id?: string };
    if (receipt.status !== 'ok' || receipt.entry_id !== entry.id) throw new Error('Recording callback did not acknowledge this entry');
  } catch (error) {
    metrics.recordingCallbackFailures.inc({ reason: controller.signal.aborted ? 'timeout' : failureReason });
    throw error;
  } finally { clearTimeout(timeout); }
}
