import { useLocation } from "react-router-dom";
import { preserveRoutedTestMode } from "../routedTestMode";

/**
 * A settings link that keeps a routed test run in test mode while moving
 * between screens. Outside `vrooli_test_mode=1` the path is returned unchanged.
 */
export function useSettingsPath(): (path: string) => string {
  const { search } = useLocation();
  return (path) => preserveRoutedTestMode(path, search);
}
