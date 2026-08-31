import { chromeTheme, StatusBarFill, useChromeContribution } from "./ChromeTheme";

/**
 * A base colour with one higher-priority contributor over it — the shape every
 * host uses: resting chrome underneath, an urgent surface claiming the notch.
 */
export function Default() {
  chromeTheme.setBase({ statusColor: "#0f172a" });
  useChromeContribution(
    { statusColor: "#3f0d0d", fillColor: "rgb(69 10 10 / 0.55)" },
    { key: "story-danger", priority: 90 },
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", ["--rcl-safe-top" as string]: "24px" }}>
      <StatusBarFill testId="services.chrome-theme-strip" />
      <output data-testid="services.chrome-theme">
        {chromeTheme.current()?.source ?? "base"}
      </output>
    </div>
  );
}
