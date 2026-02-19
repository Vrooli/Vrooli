import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "../../lib/utils";

interface PaginationProps {
  /** Current page (0-indexed) */
  page: number;
  /** Total number of items */
  total: number;
  /** Items per page */
  pageSize: number;
  /** Called when page changes */
  onPageChange: (page: number) => void;
  /** Optional className */
  className?: string;
}

export function Pagination({ page, total, pageSize, onPageChange, className }: PaginationProps) {
  const totalPages = Math.ceil(total / pageSize);
  if (totalPages <= 1) return null;

  const start = page * pageSize + 1;
  const end = Math.min((page + 1) * pageSize, total);

  return (
    <nav aria-label="Pagination" className={cn("flex items-center justify-between gap-2 text-xs text-slate-300", className)} data-testid="pagination">
      <span aria-live="polite">
        {start}–{end} of {total}
      </span>
      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={page === 0}
          className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-white/10 text-slate-300 transition-colors hover:bg-white/10 hover:text-slate-200 disabled:opacity-40 disabled:pointer-events-none"
          aria-label="Previous page"
          data-testid="pagination-prev"
        >
          <ChevronLeft className="h-4 w-4" aria-hidden="true" />
        </button>
        <span className="px-2 tabular-nums" aria-current="page">
          {page + 1} / {totalPages}
        </span>
        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages - 1}
          className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-white/10 text-slate-300 transition-colors hover:bg-white/10 hover:text-slate-200 disabled:opacity-40 disabled:pointer-events-none"
          aria-label="Next page"
          data-testid="pagination-next"
        >
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </nav>
  );
}
