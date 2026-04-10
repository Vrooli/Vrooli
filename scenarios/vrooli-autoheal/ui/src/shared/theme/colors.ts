const toRgbVar = (token: string): string => `rgb(var(${token}))`;

export const themeColors = {
  chart: {
    ok: toRgbVar("--color-status-ok"),
    warning: toRgbVar("--color-status-warning"),
    critical: toRgbVar("--color-status-critical"),
    grid: toRgbVar("--color-border-default"),
    axis: toRgbVar("--color-border-strong"),
    tick: toRgbVar("--color-text-muted"),
  },
  mermaid: {
    nodeBg: toRgbVar("--color-mermaid-node-bg"),
    clusterBg: toRgbVar("--color-mermaid-cluster-bg"),
    text: toRgbVar("--color-text-primary"),
    accent: toRgbVar("--color-accent-primary"),
    line: toRgbVar("--color-mermaid-line"),
    border: toRgbVar("--color-mermaid-border"),
  },
} as const;
