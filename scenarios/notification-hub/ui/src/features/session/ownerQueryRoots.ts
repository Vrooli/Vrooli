/** Recipient configuration summaries remain readable in the signed-out view. */
export const ANONYMOUS_SETTINGS_QUERY_ROOTS = [
  "quiet-windows",
  "escalation-chain",
  "notification-catalog",
] as const;

export const OWNER_ONLY_QUERY_ROOTS = [
  "recipient-channel-addresses",
  "device-apps",
] as const;
