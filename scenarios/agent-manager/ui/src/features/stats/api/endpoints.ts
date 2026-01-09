// Stats API endpoint constants

const API_BASE = "/api/v1/stats";

export const STATS_ENDPOINTS = {
  SUMMARY: `${API_BASE}/summary`,
  STATUS_DISTRIBUTION: `${API_BASE}/status-distribution`,
  SUCCESS_RATE: `${API_BASE}/success-rate`,
  DURATION: `${API_BASE}/duration`,
  COST: `${API_BASE}/cost`,
  RUNNERS: `${API_BASE}/runners`,
  PROFILES: `${API_BASE}/profiles`,
  MODELS: `${API_BASE}/models`,
  MODEL_RUNS: `${API_BASE}/models/runs`,
  TOOLS: `${API_BASE}/tools`,
  TOOL_MODELS: `${API_BASE}/tools/models`,
  TOOL_RUNS: `${API_BASE}/tools/runs`,
  ERRORS: `${API_BASE}/errors`,
  TIME_SERIES: `${API_BASE}/time-series`,
} as const;

export type StatsEndpoint = (typeof STATS_ENDPOINTS)[keyof typeof STATS_ENDPOINTS];
