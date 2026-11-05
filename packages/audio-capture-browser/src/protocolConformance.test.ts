import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { ATV2_MAGIC, V2_AUDIO_HEADER_BYTES } from "./protocol";
import { STREAM_MESSAGE_TYPES } from "./streamMessages";

const fixture = JSON.parse(readFileSync(resolve(process.cwd(), "src", "__fixtures__/audio-tools-atv2.json"), "utf8")) as {
  magic: string;
  headerBytes: number;
  messages: Record<string, string>;
};

describe("audio-tools ATV2 protocol conformance", () => {
  it("matches the server-owned fixture exactly", () => {
    expect({ magic: ATV2_MAGIC, headerBytes: V2_AUDIO_HEADER_BYTES, messages: STREAM_MESSAGE_TYPES }).toEqual(fixture);
  });
});
