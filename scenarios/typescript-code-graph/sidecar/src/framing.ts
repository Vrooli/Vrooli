// Line-delimited JSON framing helpers. writeMessage is the ONLY place in
// the sidecar that touches stdout; everything else logs to stderr.

import type { Response } from "./protocol.js";

export type StdoutLike = { write(chunk: string): boolean };

let target: StdoutLike = process.stdout;

/** Test-only: redirect stdout writes to a buffer. */
export function setStdoutSink(sink: StdoutLike): void {
  target = sink;
}

/** Test-only: restore the real stdout sink. */
export function resetStdoutSink(): void {
  target = process.stdout;
}

export function writeMessage(msg: Response): void {
  // JSON.stringify is sync and never embeds raw newlines in strings (they
  // become \n escapes), so the result is safe as a single line.
  const line = JSON.stringify(msg) + "\n";
  target.write(line);
}

export function parseMessage(line: string): unknown {
  return JSON.parse(line);
}
