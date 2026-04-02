/**
 * MediaLightbox — fullscreen overlay for viewing a single image or video.
 *
 * Renders via portal to document.body. Supports Escape key to close.
 * Used by evidence item cards for workflow recordings and screenshots.
 */

import { useEffect } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";

export interface MediaLightboxProps {
  isOpen: boolean;
  onClose: () => void;
  src: string;
  type: "image" | "video";
  label?: string;
}

export function MediaLightbox({ isOpen, onClose, src, type, label }: MediaLightboxProps) {
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex flex-col bg-black/95"
      onClick={onClose}
      data-testid="media-lightbox"
    >
      {/* Top bar */}
      <div
        className="flex items-center justify-between px-4 py-3"
        onClick={(e) => e.stopPropagation()}
      >
        {label && (
          <span className="truncate text-sm font-medium text-white/80">
            {label}
          </span>
        )}
        <button
          onClick={onClose}
          className="ml-auto rounded p-1 text-white/60 transition-colors hover:bg-white/10 hover:text-white"
          aria-label="Close lightbox"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Media content */}
      <div
        className="flex flex-1 items-center justify-center p-4"
        onClick={(e) => e.stopPropagation()}
      >
        {type === "video" ? (
          <video
            key={src}
            controls
            autoPlay
            src={src}
            className="max-h-full max-w-full rounded-lg"
          />
        ) : (
          <img
            key={src}
            src={src}
            alt={label ?? "Evidence"}
            className="max-h-full max-w-full rounded-lg object-contain"
          />
        )}
      </div>
    </div>,
    document.body,
  );
}
