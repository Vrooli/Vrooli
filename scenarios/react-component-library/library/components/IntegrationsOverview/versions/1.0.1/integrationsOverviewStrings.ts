/** Every visible string; {ready}, {total}, {time} and {provider} are filled in. Pass translations through `strings`. */
export interface IntegrationsOverviewStrings {
  summaryAllReady: string;
  summarySomeReady: string;
  summaryChecking: string;
  summaryError: string;
  checkedAt: string;
  refresh: string;
  refreshing: string;
  back: string;
  providedBy: string;
  statusReady: string;
  statusLimited: string;
  statusOff: string;
  statusMissing: string;
  statusChecking: string;
  statusUnknown: string;
  featuresTitle: string;
  featureUnavailable: string;
  technicalDetails: string;
  emptyGroup: string;
}

export const INTEGRATIONS_OVERVIEW_STRINGS: IntegrationsOverviewStrings = {
  summaryAllReady: "All {total} ready",
  summarySomeReady: "{ready} of {total} ready",
  summaryChecking: "Checking…",
  summaryError: "Couldn't check what's connected",
  checkedAt: "Checked {time}",
  refresh: "Check again",
  refreshing: "Checking…",
  back: "Back",
  providedBy: "Provided by {provider}",
  statusReady: "Ready",
  statusLimited: "Limited",
  statusOff: "Off",
  statusMissing: "Not installed",
  statusChecking: "Checking",
  statusUnknown: "Unknown",
  featuresTitle: "What it adds",
  featureUnavailable: "Unavailable",
  technicalDetails: "Technical details",
  emptyGroup: "Nothing here yet.",
};
