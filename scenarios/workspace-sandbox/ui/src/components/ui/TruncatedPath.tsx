import { useState } from "react";
import { cn, truncatePath } from "../../lib/utils";

interface TruncatedPathProps {
  /** The full file/directory path */
  path: string;
  /** Maximum display length before truncation (default: 35) */
  maxLength?: number;
  className?: string;
  /** Use monospace font (default: true for paths) */
  mono?: boolean;
}

/**
 * Displays a path with left-truncation. On tap/click, toggles to show the full
 * path inline. Desktop users also get a native title tooltip on hover.
 */
export function TruncatedPath({
  path,
  maxLength = 35,
  className,
  mono = true,
}: TruncatedPathProps) {
  const [expanded, setExpanded] = useState(false);
  const truncated = truncatePath(path, maxLength);
  const isTruncated = truncated !== path;

  return (
    <span
      className={cn(
        "inline-block",
        mono && "font-mono",
        isTruncated && !expanded && "truncate",
        expanded && "break-all",
        isTruncated && "cursor-pointer",
        className,
      )}
      title={isTruncated && !expanded ? path : undefined}
      onClick={
        isTruncated
          ? (e) => {
              e.stopPropagation();
              setExpanded(!expanded);
            }
          : undefined
      }
      data-testid="truncated-path"
    >
      {expanded ? path : truncated}
    </span>
  );
}
