/**
 * Formats diagnostics information into a human-readable description for issue reports
 */

import type { App, BridgeRuleReport } from '@/types';

interface RuntimeFailureEntry {
  code: string;
  label: string;
  detail?: { title: string; recommendation: string };
}

interface DiagnosticsReport {
  scenario?: string;
  checked_at?: string;
  duration_ms?: number;
  warning?: string;
  warnings?: string[];
  violations?: unknown[];
  results?: BridgeRuleReport[];
}

interface FormatDiagnosticsDescriptionParams {
  diagnosticsReport: DiagnosticsReport | null;
  runtimeFailureEntries: RuntimeFailureEntry[];
  diagnosticsRuleResults: BridgeRuleReport[];
  diagnosticsScannedFileCount: number;
  diagnosticsWarnings: string[];
  app: App | null;
  appId?: string;
}

/**
 * Formats diagnostics data into a markdown-style report description
 */
export function formatDiagnosticsDescription({
  diagnosticsReport,
  runtimeFailureEntries,
  diagnosticsRuleResults,
  diagnosticsScannedFileCount,
  diagnosticsWarnings,
  app,
  appId,
}: FormatDiagnosticsDescriptionParams): string {
  const runtimeFailures = runtimeFailureEntries.map(entry => entry.label);

  const hasDiagnosticsReport = Boolean(diagnosticsReport);
  const hasRuntimeIssues = runtimeFailures.length > 0;

  if (!hasDiagnosticsReport && !hasRuntimeIssues) {
    return '';
  }

  const hasRuleProblems = diagnosticsRuleResults.some((rule) => {
    const combinedWarnings = [rule.warning, ...(rule.warnings ?? [])]
      .map(entry => (entry ?? '').toString().trim())
      .filter(entry => entry.length > 0);
    return rule.violations.length > 0 || combinedWarnings.length > 0;
  });

  const hasScanFailure = diagnosticsScannedFileCount <= 0;
  const hasGlobalWarnings = diagnosticsWarnings.length > 0;

  if (!hasRuntimeIssues && !hasRuleProblems && !hasScanFailure && !hasGlobalWarnings) {
    return '';
  }

  const scenarioSlug = (() => {
    const candidates = [
      diagnosticsReport?.scenario,
      app?.scenario_name,
      app?.id,
      appId,
    ];
    const match = candidates
      .map(value => (value ?? '').toString().trim())
      .find(value => value.length > 0);
    return match ?? 'scenario';
  })();

  const formatTimestamp = (value: string | number | null | undefined) => {
    if (!value) {
      return null;
    }
    try {
      const date = new Date(value);
      return date.toLocaleString();
    } catch {
      return null;
    }
  };

  const lines: string[] = [];
  lines.push(`Scenario-auditor diagnostics for ${scenarioSlug}`);

  if (diagnosticsReport?.checked_at) {
    const formatted = formatTimestamp(diagnosticsReport.checked_at);
    if (formatted) {
      lines.push(`Checked at ${formatted}.`);
    }
  }

  if (diagnosticsReport) {
    const scanMeta: string[] = [];
    if (diagnosticsScannedFileCount > 0) {
      scanMeta.push(`files scanned: ${diagnosticsScannedFileCount}`);
    } else {
      scanMeta.push('no files were scanned');
    }

    if (typeof diagnosticsReport.duration_ms === 'number' && Number.isFinite(diagnosticsReport.duration_ms)) {
      scanMeta.push(`duration: ${Math.max(0, Math.round(diagnosticsReport.duration_ms))} ms`);
    }

    if (scanMeta.length > 0) {
      lines.push(`Scan summary — ${scanMeta.join(', ')}.`);
    }

    if (diagnosticsWarnings.length > 0) {
      lines.push('Scenario-auditor warnings:');
      diagnosticsWarnings.forEach((warning) => {
        lines.push(`- ${warning}`);
      });
    }

    if (diagnosticsScannedFileCount <= 0) {
      lines.push('Scenario-auditor could not inspect the UI package. Ensure the scenario depends on @vrooli/iframe-bridge, exports a bootstrap entry, and includes UI/config files in rule targets.');
    }

    const ruleDetails: string[] = [];
    diagnosticsRuleResults.forEach((rule) => {
      const trimmedRuleId = (rule.rule_id ?? '').trim();
      const ruleNameCandidate = rule.name ?? trimmedRuleId;
      const trimmedRuleName = (ruleNameCandidate ?? '').toString().trim();
      const ruleName = trimmedRuleName.length > 0 ? trimmedRuleName : 'Diagnostics rule';

      const combinedWarnings = Array.from(new Set(
        [rule.warning, ...(rule.warnings ?? [])]
          .map(entry => (entry ?? '').toString().trim())
          .filter(entry => entry.length > 0),
      ));

      const hasRuleViolations = rule.violations.length > 0;
      const hasRuleWarnings = combinedWarnings.length > 0;

      if (!hasRuleViolations && !hasRuleWarnings && diagnosticsScannedFileCount > 0) {
        return;
      }

      const headerParts = [ruleName];
      if (trimmedRuleId) {
        headerParts.push(`(${trimmedRuleId})`);
      }

      const headerLine = headerParts.join(' ').trim();
      ruleDetails.push(`- ${headerLine}`);

      if (hasRuleViolations) {
        rule.violations.forEach((violation, violationIndex) => {
          const location = violation.file_path
            ? `${violation.file_path}${typeof violation.line === 'number' ? `:${violation.line}` : ''}`
            : null;
          const locationLabel = location ? ` (${location})` : '';
          const recommendation = (violation.recommendation ?? '').trim();
          const base = `${violationIndex + 1}. ${violation.title}${locationLabel} — ${violation.description || violation.type}`;
          ruleDetails.push(`    - ${base}`);
          if (recommendation) {
            ruleDetails.push(`      Recommendation: ${recommendation}`);
          }
        });
      }

      combinedWarnings.forEach((warning, warningIndex) => {
        ruleDetails.push(`    - Warning ${warningIndex + 1}: ${warning}`);
      });

      if (trimmedRuleId) {
        ruleDetails.push(`    - Re-run: scenario-auditor scan ${scenarioSlug} --rule ${trimmedRuleId} --wait --timeout 600`);
      }
    });

    if (ruleDetails.length > 0) {
      lines.push('Rule findings:');
      lines.push(...ruleDetails);
    }
  }

  if (hasRuntimeIssues) {
    lines.push('Runtime bridge diagnostics reported issues:');
    runtimeFailures.forEach((failure) => {
      lines.push(`- ${failure}`);
    });
    lines.push('Retest by reloading the preview in App Monitor and re-running diagnostics after the iframe bridge initializes.');
  }

  if (scenarioSlug) {
    lines.push(`Full scan: scenario-auditor scan ${scenarioSlug} --wait --timeout 600`);
  }

  return lines.join('\n');
}
