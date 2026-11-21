const ROUTED_TEST_MODE_QUERY = "vrooli_test_mode";
const ROUTED_TEST_FIXTURE_QUERY = "e4_fixture";

export function isRoutedTestMode(currentSearch: string): boolean {
  return new URLSearchParams(currentSearch).get(ROUTED_TEST_MODE_QUERY) === "1";
}

/** Remove a stale disposable-test opt-in before opening the live app. */
export function livePath(path: string): string {
  const target = new URL(path, "http://notification-hub.local");
  target.searchParams.delete(ROUTED_TEST_MODE_QUERY);
  target.searchParams.delete(ROUTED_TEST_FIXTURE_QUERY);
  return `${target.pathname}${target.search}${target.hash}`;
}

/** Keep an explicit routed-test opt-in across internal SPA navigation. */
export function preserveRoutedTestMode(path: string, currentSearch: string): string {
  if (!isRoutedTestMode(currentSearch)) {
    return path;
  }

  const target = new URL(path, "http://notification-hub.local");
  target.searchParams.set(ROUTED_TEST_MODE_QUERY, "1");
  return `${target.pathname}${target.search}${target.hash}`;
}
