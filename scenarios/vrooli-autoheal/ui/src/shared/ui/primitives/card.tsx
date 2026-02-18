import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const cardVariants = cva(
  "rounded-lg border border-border-default/70 bg-surface-elevated/70 text-text-primary shadow-panel",
  {
    variants: {
      variant: {
        default: "",
        subtle: "bg-surface-elevated/40",
        interactive: "transition-colors hover:border-border-strong/80 hover:bg-surface-overlay/60",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export interface CardProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {}

export function Card({ className, variant, ...props }: CardProps) {
  return <div className={cn(cardVariants({ variant, className }))} {...props} />;
}
