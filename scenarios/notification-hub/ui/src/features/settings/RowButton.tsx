import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "../../lib/utils";

/**
 * A whole-row button that looks like `ListRow`, for rows that open an editor in
 * place rather than navigate. The chevron turns around in right-to-left text.
 */
export const RowButton = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement> & { trailing?: ReactNode }>(
  function RowButton({ children, trailing, className, type = "button", ...props }, ref) {
    return (
      <button
        ref={ref}
        type={type}
        className={cn(
          "flex min-h-14 w-full items-start gap-3 px-4 py-3 text-start text-app-foreground transition-colors duration-quick hover:bg-app-surface-muted active:bg-app-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2",
          className,
        )}
        {...props}
      >
        <div className="min-w-0 flex-1">{children}</div>
        {trailing}
        <ChevronRight size={18} className="mt-0.5 shrink-0 text-app-muted-foreground rtl:rotate-180" aria-hidden="true" />
      </button>
    );
  },
);
