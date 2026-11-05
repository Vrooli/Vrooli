import { useEffect, useRef } from 'react';

/** The source stays continuous; only its presentation is shared with replays. */
export function RecordedVideo({ source, time, controls = false }: { source: string; time?: number; controls?: boolean }) {
  const ref = useRef<HTMLVideoElement>(null);
  useEffect(() => {
    const video = ref.current;
    if (!video || time === undefined) return;
    const seek = () => {
      const target = Math.min(time, Math.max(0, video.duration - 0.001));
      if (Number.isFinite(target) && Math.abs(video.currentTime - target) > 0.001) video.currentTime = target;
    };
    if (video.readyState >= 1) seek();
    video.addEventListener('loadedmetadata', seek);
    return () => video.removeEventListener('loadedmetadata', seek);
  }, [time, source]);
  return <video ref={ref} data-bas-source-video src={source} muted preload="auto" controls={controls} playsInline className="h-full w-full object-contain" />;
}
