/**
 * Tag List Component
 *
 * Renders a list of tags with truncation support. Shows the first N tags
 * and displays a "+X more" indicator when there are additional tags.
 *
 * This component eliminates duplicate tag truncation logic across pages
 * (IdeasPage, ScenariosPage) by providing a single, reusable implementation.
 */

interface TagListProps {
  /** Array of tag strings to display */
  tags: string[];
  /** Maximum number of tags to show before truncating (default: 3) */
  maxTags?: number;
  /** Additional CSS classes for the container */
  className?: string;
}

/**
 * TagList displays tags with automatic truncation.
 *
 * @example
 * ```tsx
 * <TagList tags={["react", "typescript", "vite", "tailwind"]} maxTags={3} />
 * // Renders: react, typescript, vite, +1 more
 * ```
 */
export function TagList({ tags, maxTags = 3, className = "" }: TagListProps) {
  if (!tags || tags.length === 0) {
    return null;
  }

  const visibleTags = tags.slice(0, maxTags);
  const hiddenCount = tags.length - maxTags;

  return (
    <div className={`flex flex-wrap gap-1 ${className}`}>
      {visibleTags.map((tag) => (
        <span
          key={tag}
          className="rounded-full bg-slate-700/50 px-2 py-0.5 text-xs text-slate-400"
        >
          {tag}
        </span>
      ))}
      {hiddenCount > 0 && (
        <span className="text-xs text-slate-500">+{hiddenCount}</span>
      )}
    </div>
  );
}
