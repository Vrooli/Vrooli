import { useEffect, useMemo, useRef } from "react";
import type { JSX } from "react";

export interface AudioPlayerBarProps {
  /** Direct audio URL (e.g., audio-tools cache hit) — takes precedence over audioBytes. */
  audioUrl?: string;
  /** Raw audio bytes — wrapped as a Blob URL. */
  audioBytes?: Uint8Array;
  /** MIME type for the bytes; only used when audioBytes is provided. */
  contentType?: string;
  /** Initial playback speed (0.5–4.0). */
  speed?: number;
  /** Callback fired when playback state changes. */
  onPlayStateChange?: (state: "playing" | "paused" | "ended") => void;
}

/**
 * Generalized audio player. Wraps the native <audio> element with a Blob URL
 * lifecycle so callers can pass bytes without managing object-URL revocation.
 */
export function AudioPlayerBar(props: AudioPlayerBarProps): JSX.Element {
  const audioRef = useRef<HTMLAudioElement>(null);
  const src = useMemo(() => {
    if (props.audioUrl) return props.audioUrl;
    if (props.audioBytes && props.audioBytes.byteLength > 0) {
      // Copy into a fresh ArrayBuffer to avoid the SharedArrayBuffer
      // variance issue lib.dom's Blob constructor objects to.
      const copy = new Uint8Array(props.audioBytes.byteLength);
      copy.set(props.audioBytes);
      const blob = new Blob([copy.buffer], { type: props.contentType ?? "audio/mpeg" });
      return URL.createObjectURL(blob);
    }
    return "";
  }, [props.audioUrl, props.audioBytes, props.contentType]);

  useEffect(() => {
    return () => {
      if (src.startsWith("blob:")) URL.revokeObjectURL(src);
    };
  }, [src]);

  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;
    if (props.speed && props.speed > 0) el.playbackRate = props.speed;
  }, [props.speed]);

  // Synthesized TTS output has no captions track; consumers add subtitles
  // upstream when they know the source language. This is the documented
  // exception for audio-tools' embed surface.
  /* eslint-disable jsx-a11y/media-has-caption */
  return (
    <audio
      ref={audioRef}
      src={src}
      controls
      onPlay={() => props.onPlayStateChange?.("playing")}
      onPause={() => props.onPlayStateChange?.("paused")}
      onEnded={() => props.onPlayStateChange?.("ended")}
      className="audio-tools-embed-audio-player-bar"
    />
  );
  /* eslint-enable jsx-a11y/media-has-caption */
}
