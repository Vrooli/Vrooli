import { cva } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const tabTriggerVariants = cva(
  "flex items-center gap-2 border-b-2 px-4 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-primary/70",
  {
    variants: {
      active: {
        true: "border-accent-primary text-accent-primary",
        false: "border-transparent text-text-muted hover:text-text-primary",
      },
      size: {
        compact: "py-2",
        regular: "py-3",
      },
    },
    defaultVariants: {
      active: false,
      size: "compact",
    },
  }
);

interface TabTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean;
  size?: "compact" | "regular";
}

export function TabTrigger({ active = false, size = "compact", className, ...props }: TabTriggerProps) {
  return (
    <button
      type="button"
      className={cn(tabTriggerVariants({ active, size }), className)}
      {...props}
    />
  );
}
