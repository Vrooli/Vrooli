export const selectors = {
  app: {
    root: "sda-app-root",
    title: "sda-app-title"
  },
  layout: {
    shell: "sda-layout-shell",
    topBar: "sda-layout-topbar",
    sidebar: "sda-layout-sidebar",
    bottomNav: "sda-layout-bottom-nav",
    main: "sda-layout-main",
    navLink: (route: string) => `sda-nav-${route}`
  },
  errorBoundary: {
    root: "sda-error-boundary",
    retryButton: "sda-error-boundary-retry"
  },
  governance: {
    root: "sda-governance-root",
    triagePanel: "sda-governance-triage-panel",
    refreshButton: "sda-governance-refresh",
    summary: "sda-governance-summary",
    findingRow: (id: string) => `sda-governance-finding-${id}`,
    dependencyRow: (ecosystem: string, packageName: string) =>
      `sda-governance-dependency-${ecosystem}-${packageName}`,
    decisionForm: "sda-governance-decision-form",
    remediationForm: "sda-governance-remediation-form"
  }
} as const;
