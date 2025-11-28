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

interface VideoSectionProps {
  title?: string;
  videoUrl: string;
  thumbnailUrl?: string;
  caption?: string;
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

export function VideoSection({ title, videoUrl, thumbnailUrl, caption }: VideoSectionProps) {
  const [isPlaying, setIsPlaying] = useState(false);
  const embedUrl = getVideoEmbedUrl(videoUrl);

  if (!embedUrl) {
    console.error("[VideoSection] Invalid video URL:", videoUrl);
    return null;
  }

  return (
    <section className="py-16 px-6">
      <div className="max-w-5xl mx-auto">
        {title && (
          <h2 className="text-3xl md:text-4xl font-bold text-center mb-8 text-slate-900 dark:text-slate-50">
            {title}
          </h2>
        )}

        <div className="relative aspect-video rounded-2xl overflow-hidden shadow-2xl">
          {!isPlaying && thumbnailUrl ? (
            <button
              onClick={() => setIsPlaying(true)}
              className="relative w-full h-full group cursor-pointer"
              aria-label="Play video"
            >
              <img
                src={thumbnailUrl}
                alt="Video thumbnail"
                className="w-full h-full object-cover"
              />
              <div className="absolute inset-0 bg-black/30 group-hover:bg-black/50 transition-colors flex items-center justify-center">
                <div className="w-20 h-20 rounded-full bg-white/90 group-hover:bg-white group-hover:scale-110 transition-all flex items-center justify-center shadow-xl">
                  <Play className="w-10 h-10 text-slate-900 ml-1" fill="currentColor" />
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
          <p className="mt-6 text-center text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
            {caption}
          </p>
        )}
      </div>
    </section>
  );
}
