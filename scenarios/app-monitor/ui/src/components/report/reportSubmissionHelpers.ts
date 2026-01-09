/**
 * Helper functions for report submission workflow
 */

import type { ReportIssuePayload } from '@/services/api';
import type { ReportElementCapture } from './reportTypes';
import { REPORT_APP_LOGS_MAX_LINES } from './reportConstants';
import type { ReportLogsState } from './reportLogsReducer';

export interface ValidationResult {
  valid: boolean;
  error?: string;
}

/**
 * Validates that the report has sufficient content to submit
 */
export function validateReportSubmission(
  reportMessage: string,
  elementCaptures: ReportElementCapture[],
  includeDiagnosticsSummary: boolean,
  reportIncludeScreenshot: boolean,
  reportScreenshotData: string | null,
  canCaptureScreenshot: boolean,
  appId?: string,
): ValidationResult {
  const trimmed = reportMessage.trim();
  const captureNotes = elementCaptures.filter(capture => capture.note.trim().length > 0);

  const hasPrimaryDescription = trimmed.length > 0;
  const hasCaptureNotes = captureNotes.length > 0;

  if (!hasPrimaryDescription && !hasCaptureNotes && !includeDiagnosticsSummary) {
    return {
      valid: false,
      error: 'Add a description, include diagnostics, or add capture notes before sending.',
    };
  }

  if (!appId || appId.trim() === '') {
    return {
      valid: false,
      error: 'Unable to determine which application to report.',
    };
  }

  const includeScreenshot = reportIncludeScreenshot && canCaptureScreenshot;
  if (includeScreenshot && !reportScreenshotData) {
    return {
      valid: false,
      error: 'Capture a screenshot before sending the report.',
    };
  }

  return { valid: true };
}

/**
 * Builds the final report message from all content sources
 */
export function buildReportMessage(
  trimmedMessage: string,
  elementCaptures: ReportElementCapture[],
  diagnosticsDescription: string,
  includeDiagnosticsSummary: boolean,
): string {
  const captureNotes = elementCaptures
    .map((capture, index) => {
      const note = capture.note.trim();
      if (!note) {
        return null;
      }

      const selectorLabel = capture.metadata.selector?.trim();
      const descriptiveLabel = capture.metadata.label?.trim();
      const tagLabel = capture.metadata.tagName ? `<${capture.metadata.tagName}>` : null;
      const label = selectorLabel || descriptiveLabel || tagLabel || `Element capture ${index + 1}`;
      return `- ${label}: ${note}`;
    })
    .filter((entry): entry is string => Boolean(entry));

  const hasPrimaryDescription = trimmedMessage.length > 0;
  const hasCaptureNotes = captureNotes.length > 0;

  const sections: string[] = [];
  if (hasPrimaryDescription) {
    sections.push(trimmedMessage);
  }

  if (includeDiagnosticsSummary && diagnosticsDescription) {
    sections.push(['### Diagnostics Summary', '', diagnosticsDescription].join('\n'));
  }

  if (hasCaptureNotes) {
    sections.push(['### Element Capture Notes', '', ...captureNotes].join('\n'));
  }

  return sections.join('\n\n').trim();
}

/**
 * Builds the app logs section of the payload
 */
export function buildAppLogsPayload(
  logsState: ReportLogsState,
): Pick<ReportIssuePayload, 'logs' | 'logsTotal' | 'logsCapturedAt'> {
  const result: Pick<ReportIssuePayload, 'logs' | 'logsTotal' | 'logsCapturedAt'> = {};

  if (!logsState.include) {
    return result;
  }

  const selectedStreams = logsState.streams.filter(stream => logsState.selections[stream.key] !== false);

  if (selectedStreams.length > 0) {
    const combinedLogs: string[] = [];

    selectedStreams.forEach((stream) => {
      if (combinedLogs.length >= REPORT_APP_LOGS_MAX_LINES) {
        return;
      }

      const contextLabel = stream.type === 'background'
        ? `Background: ${stream.label}`
        : 'Lifecycle';
      combinedLogs.push(`--- ${contextLabel} ---`);

      if (stream.type === 'background' && stream.command && combinedLogs.length < REPORT_APP_LOGS_MAX_LINES) {
        combinedLogs.push(`# ${stream.command}`);
      }

      const remaining = REPORT_APP_LOGS_MAX_LINES - combinedLogs.length;
      if (remaining > 0) {
        const tail = stream.lines.slice(-remaining);
        combinedLogs.push(...tail);
      }
    });

    result.logs = combinedLogs;
    result.logsTotal = selectedStreams.reduce((total, stream) => total + stream.total, 0);
    if (logsState.fetchedAt) {
      result.logsCapturedAt = new Date(logsState.fetchedAt).toISOString();
    }
  } else if (logsState.logs.length > 0) {
    result.logs = logsState.logs;
    result.logsTotal = typeof logsState.logsTotal === 'number' ? logsState.logsTotal : logsState.logs.length;
    if (logsState.fetchedAt) {
      result.logsCapturedAt = new Date(logsState.fetchedAt).toISOString();
    }
  }

  return result;
}
