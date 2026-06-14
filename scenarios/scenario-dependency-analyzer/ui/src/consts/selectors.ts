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
  }
} as const;
