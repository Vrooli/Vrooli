// Frontend audio adoption boundary — now satisfied by @audio-tools/embed.
//
// Web-console previously re-exported in-tree hooks/voice + hooks/tts from
// here as the seam between consumer code and the audio domain. With the
// audio-tools adoption complete, this file is a one-line re-export of the
// embed package; consumer imports stay stable.

export * from "@audio-tools/embed";
