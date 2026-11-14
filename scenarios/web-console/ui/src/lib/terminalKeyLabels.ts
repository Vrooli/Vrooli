import { ARROW_DOWN_BYTES, ARROW_LEFT_BYTES, ARROW_RIGHT_BYTES, ARROW_UP_BYTES, CSI } from "./terminalKeys";

export interface InputLabel {
  kind: "text" | "key" | "unknown";
  label: string;
}

const namedSequences = new Map<string, string>([
  [ARROW_UP_BYTES, "Arrow Up"],
  [ARROW_DOWN_BYTES, "Arrow Down"],
  [ARROW_LEFT_BYTES, "Arrow Left"],
  [ARROW_RIGHT_BYTES, "Arrow Right"],
  [`${CSI}H`, "Home"],
  [`${CSI}F`, "End"],
  [`${CSI}Z`, "Shift+Tab"],
]);

const controlNames = new Map<number, string>([
  [0x09, "Tab"],
  [0x0a, "Enter"],
  [0x0d, "Enter"],
  [0x1b, "Escape"],
  [0x7f, "Backspace"],
]);

function unknownEscape(value: string): InputLabel {
  const body = value.startsWith("\x1b") ? `Esc ${value.slice(1)}` : value;
  return { kind: "unknown", label: body.length > 20 ? `${body.slice(0, 20)}…` : body };
}

/** Convert terminal bytes into labels safe for display in the pending pill. */
export function decodeInputLabel(data: string): InputLabel[] {
  const labels: InputLabel[] = [];
  let printable = "";
  const flushPrintable = () => {
    if (printable) labels.push({ kind: "text", label: printable });
    printable = "";
  };

  for (let index = 0; index < data.length;) {
    const code = data.charCodeAt(index);
    if (code === 0x1b) {
      flushPrintable();
      if (index + 1 < data.length && data[index + 1] === "[") {
        let end = index + 2;
        while (end < data.length && (data.charCodeAt(end) < 0x40 || data.charCodeAt(end) > 0x7e)) end += 1;
        if (end < data.length) {
          const sequence = data.slice(index, end + 1);
          labels.push(namedSequences.has(sequence) ? { kind: "key", label: namedSequences.get(sequence)! } : unknownEscape(sequence));
          index = end + 1;
          continue;
        }
      }
      labels.push({ kind: "key", label: "Escape" });
      index += 1;
      continue;
    }
    if (code < 0x20 || code === 0x7f) {
      flushPrintable();
      const controlName = controlNames.get(code);
      if (controlName) labels.push({ kind: "key", label: controlName });
      else if (code >= 1 && code <= 26) labels.push({ kind: "key", label: `Ctrl+${String.fromCharCode(code + 64)}` });
      else labels.push({ kind: "unknown", label: `0x${code.toString(16).padStart(2, "0")}` });
      index += 1;
      continue;
    }
    printable += data[index];
    index += 1;
  }
  flushPrintable();
  return labels;
}
