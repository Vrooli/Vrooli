import { cn } from "../lib/utils";

interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  children: React.ReactNode;
}

export function IconButton({ label, children, className, ...props }: IconButtonProps) {
  return (
    <button
      aria-label={label}
      className={cn(
        "p-2 rounded-md hover:bg-slate-800 transition-colors disabled:opacity-50",
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
}
