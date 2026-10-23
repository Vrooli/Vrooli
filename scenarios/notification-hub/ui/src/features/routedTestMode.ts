const ROUTED_TEST_MODE_QUERY = "vrooli_test_mode";

/** Keep an explicit routed-test opt-in across internal SPA navigation. */
export function preserveRoutedTestMode(path: string, currentSearch: string): string {
  if (new URLSearchParams(currentSearch).get(ROUTED_TEST_MODE_QUERY) !== "1") {
    return path;
  }

  const target = new URL(path, "http://notification-hub.local");
  target.searchParams.set(ROUTED_TEST_MODE_QUERY, "1");
  return `${target.pathname}${target.search}${target.hash}`;
}
