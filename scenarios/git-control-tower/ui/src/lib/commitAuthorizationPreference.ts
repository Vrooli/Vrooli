const SKIP_PRECOMMIT_STORAGE_KEY = "gct.commit.skipPrecommit";

export function readSkipPrecommitPreference(): boolean {
  try {
    return window.localStorage.getItem(SKIP_PRECOMMIT_STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

export function writeSkipPrecommitPreference(value: boolean): void {
  try {
    window.localStorage.setItem(SKIP_PRECOMMIT_STORAGE_KEY, value ? "1" : "0");
  } catch {
    // Storage can be unavailable in a restricted browser; the choice just
    // remains local to this page.
  }
}
