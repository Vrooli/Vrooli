import { useEffect, useState } from "react";
import { decodeWaveform, MAX_AUDIO_ANALYSIS_BYTES, MAX_AUDIO_ANALYSIS_SECONDS, readAudioBytes, readAudioTags } from "./audioAnalysis";
import type { PreviewModel } from "./types";

interface AudioAnalysis {
  peaks: number[];
  loading: boolean;
  title?: string;
  artist?: string;
  artwork?: string;
}

export function useAudioAnalysis(model: PreviewModel, duration: number): AudioAnalysis {
  const [analysis, setAnalysis] = useState<AudioAnalysis>({ peaks: [], loading: false });
  const { blobHref, sizeBytes, mimeType } = model;
  // VBR duration estimates can change as media loads. Analyze once while the
  // same source stays eligible, rather than decoding again for each estimate.
  const eligible = Number.isFinite(duration) && duration > 0 && duration <= MAX_AUDIO_ANALYSIS_SECONDS && sizeBytes > 0 && sizeBytes <= MAX_AUDIO_ANALYSIS_BYTES;
  useEffect(() => {
    if (!blobHref || !eligible) {
      setAnalysis({ peaks: [], loading: false });
      return;
    }
    const controller = new AbortController();
    let disposed = false;
    controller.signal.addEventListener("abort", () => {
      if (!disposed) setAnalysis((current) => ({ ...current, loading: false }));
    }, { once: true });
    const timeout = window.setTimeout(() => { controller.abort(); }, 20_000);
    let artworkUrl: string | undefined;
    setAnalysis({ peaks: [], loading: true });
    void (async () => {
      try {
        const bytes = await readAudioBytes(blobHref, controller.signal);
        controller.signal.throwIfAborted();
        // decodeAudioData detaches its input; tags must receive their own bytes.
        const [waveform, tags] = await Promise.allSettled([decodeWaveform(bytes.slice(0)), readAudioTags(bytes, mimeType)]);
        controller.signal.throwIfAborted();
        const metadata = tags.status === "fulfilled" ? tags.value : undefined;
        if (metadata?.artwork) artworkUrl = URL.createObjectURL(metadata.artwork);
        setAnalysis({
          peaks: waveform.status === "fulfilled" ? waveform.value : [],
          title: metadata?.title, artist: metadata?.artist, artwork: artworkUrl, loading: false,
        });
      } catch {
        if (!controller.signal.aborted) setAnalysis({ peaks: [], loading: false });
      } finally {
        window.clearTimeout(timeout);
      }
    })();
    return () => {
      disposed = true;
      controller.abort();
      window.clearTimeout(timeout);
      if (artworkUrl) URL.revokeObjectURL(artworkUrl);
    };
  }, [blobHref, mimeType, eligible]);
  return analysis;
}
