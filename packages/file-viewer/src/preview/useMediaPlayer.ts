import { useCallback, useEffect, useRef, useState } from "react";

export function useMediaPlayer<T extends HTMLMediaElement>(src: string) {
  const mediaRef = useRef<T>(null);
  const alive = useRef(true);
  const playRequest = useRef(0);
  const [duration, setDuration] = useState(0);
  const [time, setTime] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [failed, setFailed] = useState(false);
  const [playRejected, setPlayRejected] = useState(false);
  const [pendingPlay, setPendingPlay] = useState(false);
  const [rate, setRate] = useState(1);
  const [volume, setVolume] = useState(1);
  const [muted, setMuted] = useState(false);
  const [repeat, setRepeat] = useState(false);
  const [loopA, setLoopA] = useState<number | null>(null);
  const [loopB, setLoopB] = useState<number | null>(null);
  const [loopEnabled, setLoopEnabled] = useState(false);

  useEffect(() => {
    alive.current = true;
    const media = mediaRef.current;
    // React StrictMode replays effect setup after cleanup on the same element.
    // Restore the source that cleanup released before that replay.
    if (media && src && !media.getAttribute("src")) media.setAttribute("src", src);
    return () => {
      alive.current = false;
      media?.pause();
      media?.removeAttribute("src");
      media?.load();
    };
  }, [src]);

  const syncTime = useCallback(() => {
    const media = mediaRef.current;
    if (!media) return;
    if (loopEnabled && loopA !== null && loopB !== null && (media.currentTime >= loopB || media.currentTime < loopA)) media.currentTime = loopA;
    setTime(Number.isFinite(media.currentTime) ? media.currentTime : 0);
  }, [loopEnabled, loopA, loopB]);

  // Frame-level boundary checks while visible keep short A/B loops responsive;
  // native timeupdate remains the fallback when background frames are suspended.
  useEffect(() => {
    if (!playing) return;
    let frame = 0;
    let lastPaint = 0;
    const tick = (now: number) => {
      const media = mediaRef.current;
      if (media && loopEnabled && loopA !== null && loopB !== null && media.currentTime >= loopB) media.currentTime = loopA;
      if (now - lastPaint >= 100) { syncTime(); lastPaint = now; }
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => { cancelAnimationFrame(frame); };
  }, [playing, syncTime, loopEnabled, loopA, loopB]);

  const seek = (seconds: number) => {
    const media = mediaRef.current;
    if (!media || !duration || !Number.isFinite(seconds)) return;
    // Seeking outside the selection exits segment repeat, preserving markers.
    if (loopEnabled && loopA !== null && loopB !== null && (seconds < loopA || seconds >= loopB)) setLoopEnabled(false);
    media.currentTime = Math.max(0, Math.min(duration, seconds));
    setTime(media.currentTime);
  };

  const togglePlay = async () => {
    const media = mediaRef.current;
    if (!media || failed) return;
    if (!media.paused || pendingPlay) {
      playRequest.current++;
      media.pause();
      setPendingPlay(false);
      setWaiting(false);
      return;
    }
    const request = ++playRequest.current;
    setPendingPlay(true);
    setPlayRejected(false);
    if (loopEnabled && loopA !== null && loopB !== null && (media.currentTime < loopA || media.currentTime >= loopB)) media.currentTime = loopA;
    else if (media.ended) media.currentTime = 0;
    try { await media.play(); }
    catch (error) {
      if (alive.current && request === playRequest.current && !(error instanceof DOMException && error.name === "AbortError")) setPlayRejected(true);
    } finally {
      if (alive.current && request === playRequest.current) { setPendingPlay(false); setWaiting(false); }
    }
  };

  const toggleRepeat = () => { setRepeat(!repeat); setLoopEnabled(false); };
  const toggleLoop = () => {
    if (loopA === null || loopB === null) return;
    setLoopEnabled(!loopEnabled);
    setRepeat(false);
    if (!loopEnabled && mediaRef.current) { mediaRef.current.currentTime = loopA; setTime(loopA); }
  };
  const markA = () => {
    setLoopA(mediaRef.current?.currentTime ?? time);
    setLoopB(null);
    setLoopEnabled(false);
  };
  const markB = () => {
    const at = mediaRef.current?.currentTime ?? time;
    if (loopA === null || at - loopA < 0.25) return;
    setLoopB(at);
    setLoopEnabled(true);
    setRepeat(false);
    if (mediaRef.current) mediaRef.current.currentTime = loopA;
    setTime(loopA);
  };
  const clearLoop = () => { setLoopA(null); setLoopB(null); setLoopEnabled(false); };

  const events = {
    onLoadedMetadata: () => {
      const value = mediaRef.current?.duration ?? 0;
      setDuration(Number.isFinite(value) && value > 0 ? value : 0);
      setFailed(false);
    },
    onDurationChange: () => {
      const value = mediaRef.current?.duration ?? 0;
      setDuration(Number.isFinite(value) && value > 0 ? value : 0);
    },
    onTimeUpdate: syncTime,
    onSeeked: () => { setWaiting(false); syncTime(); },
    onPlay: () => { setPlaying(true); setPlayRejected(false); },
    onPlaying: () => { setPlaying(true); setWaiting(false); },
    onPause: () => { setPlaying(false); setWaiting(false); },
    onWaiting: () => { setWaiting(true); },
    onCanPlay: () => { setWaiting(false); },
    onError: () => { setFailed(true); setPlaying(false); setWaiting(false); setPendingPlay(false); },
    onEnded: () => {
      if (loopEnabled && loopA !== null) {
        const media = mediaRef.current;
        if (media) {
          media.currentTime = loopA;
          void media.play().catch(() => { if (alive.current) setPlayRejected(true); });
        }
      } else { setPlaying(false); setWaiting(false); }
    },
    onRateChange: () => { setRate(mediaRef.current?.playbackRate ?? 1); },
    onVolumeChange: () => { setVolume(mediaRef.current?.volume ?? 1); setMuted(mediaRef.current?.muted ?? false); },
  };
  return {
    mediaRef, duration, time, playing, waiting, failed, playRejected, pendingPlay,
    rate, volume, muted, repeat, loopA, loopB, loopEnabled, seek, togglePlay,
    toggleRepeat, toggleLoop, markA, markB, clearLoop, events,
    changeRate: (value: number) => { if (mediaRef.current) { mediaRef.current.playbackRate = value; setRate(value); } },
    changeVolume: (value: number) => { if (mediaRef.current) { mediaRef.current.volume = value; mediaRef.current.muted = value === 0; setVolume(mediaRef.current.volume); setMuted(mediaRef.current.muted); } },
    toggleMute: () => { if (mediaRef.current) { mediaRef.current.muted = !mediaRef.current.muted; setMuted(mediaRef.current.muted); } },
  };
}
