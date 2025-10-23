const transcriptDecoder =
  typeof TextDecoder !== "undefined" ? new TextDecoder() : null;

const MAX_CONTEXT_CHARS = 1600;
const MAX_CONTEXT_LINES = 12;

function stripControlSequences(value) {
  if (!value) return "";
  return value
    .replace(/\u001B\[[0-9;?]*[ -\/]*[@-~]/g, "")
    .replace(/\u001B\][^\u0007]*(?:\u0007|\u001B\\)/g, "")
    .replace(/\u001B[()#][0-9A-Za-z]/g, "")
    .replace(/[\u0000-\u0008\u000B-\u001F\u007F]/g, "")
    .replace(/\r/g, "");
}

function decodeTranscriptEntry(entry) {
  if (!entry || typeof entry.data !== "string" || entry.data.length === 0) {
    return "";
  }

  const normalizeBase64 = (value) => value.replace(/\s+/g, "");

  if (entry.encoding === "base64") {
    if (!transcriptDecoder) return "";
    try {
      const bytes = Uint8Array.from(atob(normalizeBase64(entry.data)), (c) =>
        c.charCodeAt(0),
      );
      return transcriptDecoder.decode(bytes);
    } catch (_error) {
      return "";
    }
  }

  if (entry.encoding === "utf-8") {
    if (transcriptDecoder) {
      try {
        const bytes = Uint8Array.from(atob(normalizeBase64(entry.data)), (c) =>
          c.charCodeAt(0),
        );
        return transcriptDecoder.decode(bytes);
      } catch (_error) {
        // fall through to raw string
      }
    }
  }

  return entry.data;
}

export function extractTerminalContext(tab) {
  if (!tab || !Array.isArray(tab.transcript) || tab.transcript.length === 0) {
    return [];
  }

  const collected = [];
  let charBudget = MAX_CONTEXT_CHARS;

  for (
    let i = tab.transcript.length - 1;
    i >= 0 && charBudget > 0 && collected.length < MAX_CONTEXT_LINES;
    i -= 1
  ) {
    const entry = tab.transcript[i];
    if (!entry || typeof entry.data !== "string") continue;
    const direction = entry.direction || "stdout";
    if (direction && !["stdout", "stderr", "stdin"].includes(direction)) {
      continue;
    }

    const decoded = decodeTranscriptEntry(entry);
    if (!decoded) continue;
    const sanitized = stripControlSequences(decoded);
    if (!sanitized) continue;

    const entryLines = sanitized
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0);

    if (entryLines.length === 0) {
      continue;
    }

    if (direction === "stdin") {
      for (let j = 0; j < entryLines.length; j += 1) {
        entryLines[j] = `$ ${entryLines[j]}`;
      }
    }

    for (
      let j = entryLines.length - 1;
      j >= 0 && charBudget > 0 && collected.length < MAX_CONTEXT_LINES;
      j -= 1
    ) {
      const line = entryLines[j];
      if (!line) continue;
      charBudget -= line.length;
      collected.push(line);
    }
  }

  return collected.reverse();
}
