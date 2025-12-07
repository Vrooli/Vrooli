import { Play } from "lucide-react";
import { useState } from "react";

/**
 * VideoSection component [REQ:DESIGN-VIDEO]
 *
 * Renders a video demo section with:
 * - YouTube/Vimeo embed support
 * - Custom thumbnail with play button overlay
 * - Responsive iframe for video playback
 * - Caption/description text
 *
 * Supports both direct YouTube/Vimeo URLs and embed URLs.
 * Automatically extracts video IDs and generates proper embed URLs.
 */

export interface VideoSectionContent {
  title?: string;
  videoUrl?: string;
  thumbnailUrl?: string;
  caption?: string;
}

interface VideoSectionProps extends VideoSectionContent {
  content?: VideoSectionContent;
}

/**
 * Extracts video ID from YouTube or Vimeo URL
 * Handles various URL formats:
 * - youtube.com/watch?v=ID
 * - youtu.be/ID
 * - youtube.com/embed/ID
 * - vimeo.com/ID
 * - vimeo.com/video/ID
 */
function getVideoEmbedUrl(url: string): string | null {
  // YouTube patterns
  const youtubeMatch = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})/);
  if (youtubeMatch) {
    return `https://www.youtube.com/embed/${youtubeMatch[1]}`;
  }

  // Vimeo patterns
  const vimeoMatch = url.match(/vimeo\.com\/(?:video\/)?(\d+)/);
  if (vimeoMatch) {
    return `https://player.vimeo.com/video/${vimeoMatch[1]}`;
  }

  return null;
}

export function VideoSection(props: VideoSectionProps) {
  const resolved = props.content ?? props;
  const title = resolved.title;
  const thumbnailUrl = resolved.thumbnailUrl;
  const caption = resolved.caption;
  const rawVideoUrl = typeof resolved.videoUrl === 'string' ? resolved.videoUrl.trim() : '';

  if (!rawVideoUrl) {
    return null;
  }

  const [isPlaying, setIsPlaying] = useState(false);
  const embedUrl = getVideoEmbedUrl(rawVideoUrl);

  if (!embedUrl) {
    console.error("[VideoSection] Invalid video URL:", rawVideoUrl);
    return null;
  }

  return (
    <section className="bg-[#07090F] py-20 px-6">
      <div className="mx-auto max-w-5xl">
        {title && (
          <h2 className="mb-8 text-center text-3xl font-semibold text-white md:text-4xl">{title}</h2>
        )}

        <div className="relative aspect-video overflow-hidden rounded-[32px] border border-white/10 bg-[#0F172A] shadow-[0_25px_50px_rgba(0,0,0,0.45)]">
          {!isPlaying && thumbnailUrl ? (
            <button
              onClick={() => setIsPlaying(true)}
              className="group relative h-full w-full cursor-pointer"
              aria-label="Play video"
            >
              <img
                src={thumbnailUrl}
                alt="Video thumbnail"
                className="h-full w-full object-cover"
              />
              <div className="absolute inset-0 flex items-center justify-center bg-black/40 transition-colors group-hover:bg-black/60">
                <div className="flex h-20 w-20 items-center justify-center rounded-full bg-white text-slate-900 shadow-2xl transition-transform group-hover:scale-110">
                  <Play className="ml-1 h-10 w-10" fill="currentColor" />
                </div>
              </div>
            </button>
          ) : (
            <iframe
              src={`${embedUrl}?autoplay=${isPlaying ? '1' : '0'}`}
              className="w-full h-full"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
              title={title || "Video demo"}
            />
          )}
        </div>

        {caption && (
          <p className="mx-auto mt-6 max-w-2xl text-center text-sm text-slate-300">{caption}</p>
        )}
      </div>
    </section>
  );
}
