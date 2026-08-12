/**
 * @vrooliComponentSource react-component-library:Tokens
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 87e1db9e-2394-48fe-9e56-a302efc09f0e
 * @vrooliComponentAppliedAt 2026-08-12T12:59:49Z
 * @vrooliComponentSourceSha256 00f634de4489f6d8b01d18f688b4b665da39125beabcfebb6a16e00259fb4a29
 * @vrooliComponentDriftHash ba44f2db85cc9f763fb597ff2735ee7b2c955b49a565c4672809f22358690cd8
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const TOKEN_RAMPS = {
  space: [
    "var(--space-3xs)",
    "var(--space-2xs)",
    "var(--space-xs)",
    "var(--space-sm)",
    "var(--space-md)",
    "var(--space-lg)",
    "var(--space-xl)",
    "var(--space-2xl)",
  ],
  text: [
    "var(--text-display)",
    "var(--text-title)",
    "var(--text-heading)",
    "var(--text-body)",
    "var(--text-label)",
    "var(--text-caption)",
    "var(--text-code)",
    "var(--text-overline)",
  ],
  radius: [
    "var(--radius-none)",
    "var(--radius-sm)",
    "var(--radius-md)",
    "var(--radius-lg)",
    "var(--radius-xl)",
    "var(--radius-pill)",
  ],
  elevation: [
    "var(--elev-flat)",
    "var(--elev-raised)",
    "var(--elev-floating)",
    "var(--elev-overlay)",
  ],
  layer: [
    "var(--layer-base)",
    "var(--layer-raised)",
    "var(--layer-popover)",
    "var(--layer-modal)",
    "var(--layer-toast)",
  ],
  motion: ["var(--dur-instant)", "var(--dur-fast)", "var(--dur-normal)", "var(--dur-slow)"],
} as const;

export const SEMANTIC_TOKENS = {
  background: "var(--app-background)",
  foreground: "var(--app-foreground)",
  surface: "var(--app-surface)",
  surfaceMuted: "var(--app-surface-muted)",
  border: "var(--app-border)",
  muted: "var(--app-muted-foreground)",
  primary: "var(--app-primary)",
  primaryForeground: "var(--app-primary-foreground)",
  accent: "var(--app-accent)",
  success: "var(--app-success)",
  warning: "var(--app-warning)",
  danger: "var(--app-danger)",
  info: "var(--app-info)",
  focus: "var(--app-focus)",
} as const;

export const COMPONENT_TOKENS = {
  controlHeight: "var(--control-height)",
  controlRadius: "var(--control-radius)",
  controlPadding: "var(--control-padding)",
  panelRadius: "var(--panel-radius)",
  panelPadding: "var(--panel-padding)",
  focusRing: "var(--focus-ring)",
} as const;

export type TextStyle = keyof typeof TEXT_STYLES;
export const TEXT_STYLES = {
  display: "var(--text-display)",
  title: "var(--text-title)",
  heading: "var(--text-heading)",
  body: "var(--text-body)",
  label: "var(--text-label)",
  caption: "var(--text-caption)",
  code: "var(--text-code)",
  overline: "var(--text-overline)",
} as const;

export const tokens = {
  ramps: TOKEN_RAMPS,
  semantic: SEMANTIC_TOKENS,
  component: COMPONENT_TOKENS,
  text: TEXT_STYLES,
} as const;
