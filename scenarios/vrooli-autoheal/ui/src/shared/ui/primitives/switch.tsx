import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const switchTrackVariants = cva(
  "relative inline-flex shrink-0 cursor-pointer items-center rounded-pill border border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-primary/70 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      tone: {
        neutral: "",
        primary: "",
        success: "",
        warning: "",
        danger: "",
      },
      size: {
        sm: "h-5 w-10",
        default: "h-6 w-12",
      },
      checked: {
        true: "",
        false: "bg-border-strong",
      },
    },
    compoundVariants: [
      { tone: "neutral", checked: true, className: "bg-border-strong" },
      { tone: "primary", checked: true, className: "bg-accent-primary" },
      { tone: "success", checked: true, className: "bg-accent-success" },
      { tone: "warning", checked: true, className: "bg-accent-warning" },
      { tone: "danger", checked: true, className: "bg-accent-danger" },
    ],
    defaultVariants: {
      tone: "primary",
      size: "default",
      checked: false,
    },
  }
);

const switchThumbVariants = cva(
  "pointer-events-none absolute left-1 rounded-full bg-surface-elevated transition-transform",
  {
    variants: {
      size: {
        sm: "h-4 w-4",
        default: "h-4 w-4",
      },
      checked: {
        true: "",
        false: "",
      },
    },
    compoundVariants: [
      { size: "sm", checked: true, className: "translate-x-5" },
      { size: "default", checked: true, className: "translate-x-6" },
    ],
    defaultVariants: {
      size: "default",
      checked: false,
    },
  }
);

export interface SwitchProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "onChange">,
    VariantProps<typeof switchTrackVariants> {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

export function Switch({
  checked,
  onCheckedChange,
  tone,
  size,
  className,
  ...props
}: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      className={cn(switchTrackVariants({ tone, size, checked, className }))}
      onClick={() => onCheckedChange(!checked)}
      {...props}
    >
      <span className={cn(switchThumbVariants({ size, checked }))} />
    </button>
  );
}
