/** Every visible string; {app} and {device} are filled in. Pass translations through `strings`. */
export interface AppNotificationStrings {
  title: string;
  loading: string;
  loadError: string;
  retry: string;
  installFirst: string;
  unsupported: string;
  ineligible: string;
  offDescription: string;
  turnOn: string;
  turningOn: string;
  chooseTitle: string;
  chooseDescription: string;
  bestMatch: string;
  alsoHere: string;
  standaloneLabel: string;
  standaloneDescription: string;
  confirm: string;
  cancel: string;
  onDescription: string;
  onStandaloneDescription: string;
  turnOff: string;
  turningOff: string;
  /** @deprecated No longer shown; the panel re-checks when the app returns to the foreground. */
  checkAgain: string;
  statusOn: string;
  statusOff: string;
  statusPaused: string;
  statusUnavailable: string;
  mutedDescription: string;
  pausedDescription: string;
  lastSeen: string;
  categoriesTitle: string;
  categoriesDescription: string;
  categoriesError: string;
  minimumUrgency: string;
  urgencyLow: string;
  urgencyNormal: string;
  urgencyHigh: string;
  urgencyCritical: string;
  saveError: string;
}

export const APP_NOTIFICATION_STRINGS: AppNotificationStrings = {
  title: "Notifications in this app",
  loading: "Checking notifications…",
  loadError: "Notification settings could not be loaded.",
  retry: "Try again",
  installFirst:
    "To get notifications in this app, add it to your Home Screen and open it from there.",
  unsupported:
    "This browser can't show notifications for this app. Notifications still arrive in Chimesail.",
  ineligible:
    "Notifications in this app aren't available yet. {reason} Notifications still arrive in Chimesail.",
  offDescription:
    "{app} notifications arrive in Chimesail on this device. Get them here instead, with this app's name and icon.",
  turnOn: "Get notifications in this app",
  turningOn: "Turning on…",
  chooseTitle: "Which device is this?",
  chooseDescription:
    "{app} notifications will appear here instead of in Chimesail on this device. Other devices keep Chimesail.",
  bestMatch: "Best match",
  alsoHere: "Also here: {apps}",
  standaloneLabel: "A device without Chimesail",
  standaloneDescription: "Only this app will show {app} notifications here.",
  confirm: "Use this app here",
  cancel: "Cancel",
  onDescription: "{app} notifications appear in this app on this {device} instead of in Chimesail.",
  onStandaloneDescription: "{app} notifications appear in this app on this device.",
  turnOff: "Use Chimesail instead",
  turningOff: "Switching…",
  checkAgain: "Check again",
  statusOn: "On",
  statusOff: "Off",
  statusPaused: "Paused",
  statusUnavailable: "Unavailable",
  mutedDescription:
    "Notifications for {app} are turned off in this device's settings, so they arrive in Chimesail for now. Turn them on in Settings → Notifications → {app}; this updates when you come back.",
  pausedDescription: "{reason} {app} notifications arrive in Chimesail until then.",
  lastSeen: "Last seen {date}",
  categoriesTitle: "What to notify you about",
  categoriesDescription: "These choices apply everywhere {app} notifications appear.",
  categoriesError: "Notification types could not be loaded.",
  minimumUrgency: "Minimum urgency",
  urgencyLow: "Low",
  urgencyNormal: "Normal",
  urgencyHigh: "High",
  urgencyCritical: "Critical",
  saveError: "That change was not saved. Try again.",
};
