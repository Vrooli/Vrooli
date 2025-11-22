import { cn } from "../../lib/utils";

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {}

export function Select({ className, ...props }: SelectProps) {
  return (
    <select
      className={cn(
        "flex h-10 w-full items-center justify-between rounded-lg border border-white/20 bg-white/5 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500",
        "focus:border-white/40 focus:outline-none focus:ring-2 focus:ring-white/20",
        "disabled:cursor-not-allowed disabled:opacity-50",
        "hover:bg-white/10 transition-colors",
        "[&>option]:bg-slate-900 [&>option]:text-slate-100",
        className
      )}
      {...props}
    />
  );
}
