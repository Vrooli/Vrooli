/**
 * Every visible string; {ready}, {total}, {time} and {provider} are filled in.
 * Translate through the host's library strings provider (keys
 * `integrationsOverview.<name>`) or pass them through `strings`.
 */
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
  viewBy: string;
  byCapability: string;
  byDependency: string;
  installedVersion: string;
  providerId: string;
  capabilityIds: string;
  location: string;
  bundled: string;
  onMachine: string;
  version: string;
  answering: string;
  health: string;
  lastChecked: string;
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
  viewBy: "View integrations by",
  byCapability: "By capability",
  byDependency: "By dependency",
  installedVersion: "Version {version}",
  providerId: "Provider ID",
  capabilityIds: "Capabilities",
  location: "Location",
  bundled: "Bundled with this app",
  onMachine: "On this machine",
  version: "Installed version",
  answering: "Answering requests",
  health: "Health",
  lastChecked: "Last checked",
};
