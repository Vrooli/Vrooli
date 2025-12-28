import { useState, type ReactNode, type ComponentPropsWithoutRef } from "react";
import { ExternalLink, Loader2 } from "lucide-react";
import { Tooltip } from "../../ui/tooltip";
import { useLinkPreview } from "../hooks/useLinkPreview";

interface LinkWithPreviewProps extends ComponentPropsWithoutRef<"a"> {
  href?: string;
  children?: ReactNode;
}

/**
 * Link component with hover preview showing OpenGraph metadata.
 * Opens in new tab, shows preview in tooltip on hover.
 */
export function LinkWithPreview({ href, children, ...props }: LinkWithPreviewProps) {
  const [hovered, setHovered] = useState(false);
  const { preview, isLoading, fetch } = useLinkPreview(href || "");

  const handleMouseEnter = () => {
    setHovered(true);
    if (href) {
      fetch();
    }
  };

  const handleMouseLeave = () => {
    setHovered(false);
  };

  // Parse domain from URL for fallback display
  const domain = (() => {
    try {
      const url = new URL(href || "");
      return url.hostname;
    } catch {
      return href;
    }
  })();

  // Build tooltip content
  const tooltipContent = (
    <div className="max-w-xs">
      {isLoading ? (
        <div className="flex items-center gap-2 p-2">
          <Loader2 className="h-4 w-4 animate-spin text-slate-400" />
          <span className="text-sm text-slate-400">Loading preview...</span>
        </div>
      ) : preview && (preview.title || preview.description) ? (
        <div className="p-2 space-y-2">
          {preview.favicon && (
            <img
              src={preview.favicon}
              alt=""
              className="h-4 w-4 inline-block mr-2"
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = "none";
              }}
            />
          )}
          {preview.site_name && (
            <span className="text-xs text-slate-400">{preview.site_name}</span>
          )}
          {preview.title && (
            <p className="text-sm font-medium text-slate-200 line-clamp-2">
              {preview.title}
            </p>
          )}
          {preview.description && (
            <p className="text-xs text-slate-400 line-clamp-3">
              {preview.description}
            </p>
          )}
          {preview.image && (
            <img
              src={preview.image}
              alt=""
              className="rounded max-h-24 w-full object-cover"
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = "none";
              }}
            />
          )}
        </div>
      ) : (
        <div className="p-2 flex items-center gap-2">
          <ExternalLink className="h-3 w-3 text-slate-400" />
          <span className="text-sm text-slate-400">{domain}</span>
        </div>
      )}
    </div>
  );

  return (
    <Tooltip content={tooltipContent} side="top">
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="text-indigo-400 hover:text-indigo-300 underline inline-flex items-center gap-1"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        {...props}
      >
        {children}
        <ExternalLink className="h-3 w-3 inline-block" />
      </a>
    </Tooltip>
  );
}
