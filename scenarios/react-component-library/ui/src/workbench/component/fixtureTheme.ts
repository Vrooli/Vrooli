/**
 * The workbench is a Nocturne surface, so a fixture that renders on a white
 * island is a lie about the component's surroundings. These are the same
 * BaseStyles token names the harness already understands; the specimen inherits
 * the surface the operator is actually looking at.
 */
export type FixtureSurface = "dark" | "light";

export interface FixtureTheme {
  id: string;
  tokens: Record<string, string>;
  resolved: FixtureSurface;
}

const DARK_TOKENS: Record<string, string> = {
  "--color-background": "#0b0a0f",
  "--color-shell": "#0b0a0f",
  "--color-surface": "#141219",
  "--color-surface-muted": "#1a1721",
  "--color-surface-raised": "#211c27",
  "--color-surface-sunken": "#0d0c12",
  "--color-surface-alt": "#1a1721",
  "--color-field": "#1a1721",
  "--color-foreground": "#f7f1e8",
  "--color-muted-foreground": "#a79b92",
  "--color-border": "#2b2531",
  "--color-border-strong": "#3a3340",
  "--color-primary": "#ffb35c",
  "--color-primary-foreground": "#1a1108",
  "--color-primary-hover": "#ffc785",
  "--color-focus": "#ffb35c",
  "--color-danger": "#ff8e8e",
  "--color-success": "#82e0b0",
  "--color-warning": "#ffcc80",
  "--color-info": "#8fd8ff",
  "--color-accent": "#8fd8ff",
};

const LIGHT_TOKENS: Record<string, string> = {
  "--color-background": "#f8fafc",
  "--color-shell": "#f8fafc",
  "--color-surface": "#ffffff",
  "--color-surface-muted": "#f1f5f9",
  "--color-surface-raised": "#ffffff",
  "--color-surface-sunken": "#eef2f7",
  "--color-surface-alt": "#f1f5f9",
  "--color-field": "#ffffff",
  "--color-foreground": "#0f172a",
  "--color-muted-foreground": "#64748b",
  "--color-border": "#cbd5e1",
  "--color-border-strong": "#94a3b8",
  "--color-primary": "#0891b2",
  "--color-primary-foreground": "#ffffff",
  "--color-primary-hover": "#0e7490",
  "--color-focus": "#0891b2",
  "--color-danger": "#dc2626",
  "--color-success": "#16803c",
  "--color-warning": "#a15c00",
  "--color-info": "#0369a1",
  "--color-accent": "#0891b2",
};

export function fixtureTheme(surface: FixtureSurface): FixtureTheme {
  return surface === "light"
    ? { id: "workbench-light", tokens: LIGHT_TOKENS, resolved: "light" }
    : { id: "workbench-dark", tokens: DARK_TOKENS, resolved: "dark" };
}
