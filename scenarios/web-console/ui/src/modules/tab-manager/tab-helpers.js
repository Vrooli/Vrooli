// @ts-check

/**
 * @typedef {import("../types.d.ts").TerminalTab} TerminalTab
 */

export const TAB_DEFAULT_LABEL_PREFIX = "Term";
const RANDOM_LABEL_ALPHABET = "abcdefghijklmnopqrstuvwxyz";
const RANDOM_LABEL_LENGTH = 3;
const RANDOM_LABEL_ATTEMPTS = 10;

export const TAB_SESSION_STATE_META = {
  attached: {
    icon: null,
    label: "",
  },
  starting: {
    icon: "loader-2",
    label: "Session starting",
    extraClass: "tab-status-loading",
  },
  closing: {
    icon: "power",
    label: "Stopping session",
    extraClass: "tab-status-closing",
  },
  detached: {
    icon: "unlink",
    label: "Detached from session",
    extraClass: "tab-status-detached",
  },
  none: {
    icon: "minus",
    label: "No session attached",
    extraClass: "tab-status-empty",
  },
};

function resolveCrypto() {
  const candidate =
    typeof globalThis !== "undefined" ? globalThis.crypto : null;
  if (candidate && typeof candidate.getRandomValues === "function") {
    return candidate;
  }
  if (
    typeof window !== "undefined" &&
    window.crypto &&
    typeof window.crypto.getRandomValues === "function"
  ) {
    return window.crypto;
  }
  return null;
}

function generateRandomLetters(length = RANDOM_LABEL_LENGTH) {
  const alphabetLength = RANDOM_LABEL_ALPHABET.length;
  const buffer = [];
  const cryptoObj = resolveCrypto();
  if (cryptoObj) {
    const values = new Uint8Array(length);
    cryptoObj.getRandomValues(values);
    for (let index = 0; index < length; index += 1) {
      buffer.push(RANDOM_LABEL_ALPHABET[values[index] % alphabetLength]);
    }
  } else {
    for (let index = 0; index < length; index += 1) {
      const randomIndex = Math.floor(Math.random() * alphabetLength);
      buffer.push(RANDOM_LABEL_ALPHABET[randomIndex]);
    }
  }
  return buffer.join("");
}

/**
 * Generate a unique default label for a tab.
 * @param {Set<string>} [existingLabels]
 * @returns {string}
 */
export function generateDefaultTabLabel(existingLabels = new Set()) {
  const labels = existingLabels instanceof Set ? existingLabels : new Set();
  for (let attempt = 0; attempt < RANDOM_LABEL_ATTEMPTS; attempt += 1) {
    const candidate = `${TAB_DEFAULT_LABEL_PREFIX} ${generateRandomLetters()}`;
    if (!labels.has(candidate)) {
      return candidate;
    }
  }
  return `${TAB_DEFAULT_LABEL_PREFIX} ${generateRandomLetters()}`;
}
