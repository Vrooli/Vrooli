interface ShowMoreButtonProps {
  shown: number;
  total: number;
  onShowMore: () => void;
  testId?: string;
}

/** The "Show more (N of M)" control for a list bounded by useBoundedList. */
export function ShowMoreButton({ shown, total, onShowMore, testId }: ShowMoreButtonProps) {
  if (shown >= total) return null;
  return (
    <button
      type="button"
      data-testid={testId}
      className="min-h-11 w-full rounded-md border border-border bg-background px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      onClick={onShowMore}
    >
      Show more ({shown} of {total})
    </button>
  );
}
