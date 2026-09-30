// BAS uses this explicit, opt-in query parameter to observe the loading state
// before the public configuration request begins. It is intentionally ignored
// unless the test workflow requests it, so production navigation is unchanged.
export function waitForLandingWorkflowLoadingState(): Promise<void> {
  if (typeof window === 'undefined' || new URLSearchParams(window.location.search).get('e2e_loading') !== '1') {
    return Promise.resolve();
  }
  return new Promise((resolve) => window.setTimeout(resolve, 2000));
}
