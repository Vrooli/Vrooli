// CSV export utilities for trend data
// [REQ:UI-EVENTS-001]

import { CheckTrend, Incident } from "./api";

interface TrendExportData {
  checkTrends: CheckTrend[];
  incidents: Incident[];
  windowHours: number;
  uptimePercentage: number;
}

/**
 * Escapes a value for CSV format
 */
function escapeCSV(value: string | number | undefined): string {
  if (value === undefined || value === null) return "";
  const str = String(value);
  // If contains comma, newline, or quote, wrap in quotes and escape quotes
  if (str.includes(",") || str.includes("\n") || str.includes('"')) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

/**
 * Converts an array of objects to CSV format
 */
function toCSV<T>(
  data: T[],
  headers: { key: keyof T; label: string }[]
): string {
  const headerRow = headers.map((h) => escapeCSV(h.label)).join(",");
  const dataRows = data.map((row) =>
    headers.map((h) => escapeCSV(row[h.key] as string | number)).join(",")
  );
  return [headerRow, ...dataRows].join("\n");
}

/**
 * Triggers a browser download of a file
 */
function downloadFile(content: string, filename: string, mimeType = "text/csv"): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Exports trend data to CSV and triggers download
 */
export function exportTrendDataToCSV(data: TrendExportData): void {
  const timestamp = new Date().toISOString().split("T")[0];

  // Generate check trends CSV
  const trendsCSV = toCSV(data.checkTrends, [
    { key: "checkId", label: "Check ID" },
    { key: "total", label: "Total Checks" },
    { key: "ok", label: "OK" },
    { key: "warning", label: "Warning" },
    { key: "critical", label: "Critical" },
    { key: "uptimePercent", label: "Uptime %" },
    { key: "currentStatus", label: "Current Status" },
    { key: "lastChecked", label: "Last Checked" },
  ]);

  // Generate incidents CSV
  const incidentsCSV = toCSV(data.incidents, [
    { key: "timestamp", label: "Timestamp" },
    { key: "checkId", label: "Check ID" },
    { key: "fromStatus", label: "From Status" },
    { key: "toStatus", label: "To Status" },
    { key: "message", label: "Message" },
  ]);

  // Combine with summary header
  const summary = [
    "# Vrooli Autoheal Trend Export",
    `# Generated: ${new Date().toISOString()}`,
    `# Time Window: ${data.windowHours} hours`,
    `# Overall Uptime: ${data.uptimePercentage.toFixed(2)}%`,
    "",
    "## Check Trends",
    trendsCSV,
    "",
    "## Status Transitions (Incidents)",
    incidentsCSV,
  ].join("\n");

  downloadFile(summary, `autoheal-trends-${timestamp}.csv`);
}

/**
 * Exports a single check's history to CSV
 */
export function exportCheckHistoryToCSV(
  checkId: string,
  history: { timestamp: string; status: string; message: string }[]
): void {
  const timestamp = new Date().toISOString().split("T")[0];

  const csv = toCSV(history, [
    { key: "timestamp", label: "Timestamp" },
    { key: "status", label: "Status" },
    { key: "message", label: "Message" },
  ]);

  downloadFile(csv, `autoheal-${checkId}-history-${timestamp}.csv`);
}
