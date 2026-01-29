/**
 * LinkPreviewTooltip - Tooltip component showing OG metadata for links.
 */

import { useEffect, useState, useRef } from 'react'
import { ExternalLink, Globe, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { fetchLinkPreview } from '@/lib/api'

interface OGMetadata {
  url: string
  title: string
  description: string
  image: string
  siteName: string
  favicon: string
  type?: string
}

interface LinkPreviewTooltipProps {
  url: string
  position: { x: number; y: number }
  onClose: () => void
}

/**
 * Link preview tooltip showing OG metadata.
 */
export function LinkPreviewTooltip({ url, position, onClose }: LinkPreviewTooltipProps) {
  const [metadata, setMetadata] = useState<OGMetadata | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState(false)
  const tooltipRef = useRef<HTMLDivElement>(null)

  // Fetch metadata on mount
  useEffect(() => {
    let cancelled = false

    const loadMetadata = async () => {
      setIsLoading(true)
      setError(false)

      const data = await fetchLinkPreview(url)

      if (!cancelled) {
        if (data) {
          setMetadata({
            url,
            title: data.title ?? '',
            description: data.description ?? '',
            image: data.image ?? '',
            siteName: data.site_name ?? '',
            favicon: data.favicon ?? '',
          })
        } else {
          setError(true)
        }
        setIsLoading(false)
      }
    }

    // Debounce the fetch
    const timer = setTimeout(() => {
      void loadMetadata()
    }, 300)

    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [url])

  // Calculate tooltip position to stay within viewport
  const [adjustedPosition, setAdjustedPosition] = useState(position)

  useEffect(() => {
    if (tooltipRef.current) {
      const rect = tooltipRef.current.getBoundingClientRect()
      const viewportWidth = window.innerWidth
      const viewportHeight = window.innerHeight

      let x = position.x
      let y = position.y + 20 // Below the link

      // Adjust horizontal position
      if (x + rect.width > viewportWidth - 10) {
        x = viewportWidth - rect.width - 10
      }
      if (x < 10) {
        x = 10
      }

      // Adjust vertical position
      if (y + rect.height > viewportHeight - 10) {
        y = position.y - rect.height - 10 // Above the link
      }

      setAdjustedPosition({ x, y })
    }
  }, [position, metadata])

  // Get display URL (shortened)
  const displayUrl = new URL(url).hostname

  return (
    <div
      ref={tooltipRef}
      className={cn(
        'fixed z-50 w-80 max-w-[90vw]',
        'bg-popover border border-border rounded-lg shadow-xl',
        'animate-in fade-in-0 zoom-in-95 duration-150'
      )}
      style={{
        left: adjustedPosition.x,
        top: adjustedPosition.y,
      }}
      onMouseLeave={onClose}
    >
      {isLoading ? (
        <div className="flex items-center justify-center p-4 gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span className="text-sm">Loading preview...</span>
        </div>
      ) : error || !metadata ? (
        <div className="p-3">
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-sm text-primary hover:underline"
          >
            <ExternalLink className="h-4 w-4" />
            <span className="truncate">{url}</span>
          </a>
        </div>
      ) : (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="block hover:bg-muted/50 transition-colors rounded-lg overflow-hidden"
        >
          {/* Image */}
          {metadata.image && (
            <div className="relative h-32 bg-muted overflow-hidden">
              <img
                src={metadata.image}
                alt=""
                className="w-full h-full object-cover"
                onError={(e) => {
                  e.currentTarget.style.display = 'none'
                }}
              />
            </div>
          )}

          {/* Content */}
          <div className="p-3">
            {/* Site info */}
            <div className="flex items-center gap-2 mb-2">
              {metadata.favicon ? (
                <img
                  src={metadata.favicon}
                  alt=""
                  className="w-4 h-4"
                  onError={(e) => {
                    e.currentTarget.style.display = 'none'
                    const fallback = e.currentTarget.nextElementSibling
                    if (fallback) (fallback as HTMLElement).style.display = 'block'
                  }}
                />
              ) : (
                <Globe className="h-4 w-4 text-muted-foreground" />
              )}
              <Globe className="h-4 w-4 text-muted-foreground hidden" />
              <span className="text-xs text-muted-foreground truncate">
                {metadata.siteName || displayUrl}
              </span>
            </div>

            {/* Title */}
            {metadata.title && (
              <h4 className="text-sm font-medium text-foreground line-clamp-2 mb-1">
                {metadata.title}
              </h4>
            )}

            {/* Description */}
            {metadata.description && (
              <p className="text-xs text-muted-foreground line-clamp-2">
                {metadata.description}
              </p>
            )}
          </div>
        </a>
      )}
    </div>
  )
}
