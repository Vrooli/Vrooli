import { cn } from "../../../lib/utils";
import { Card, type CardProps } from "../primitives";
import { cva, type VariantProps } from "class-variance-authority";

export function Panel({ className, ...props }: CardProps) {
  return <Card className={cn("p-4", className)} {...props} />;
}

export function PanelHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mb-3 flex items-center justify-between gap-2", className)} {...props} />;
}

export function PanelTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return <h3 className={cn("text-sm font-medium text-text-primary", className)} {...props} />;
}

const noticeVariants = cva("rounded-lg border p-3", {
  variants: {
    tone: {
      info: "border-accent-primary/20 bg-accent-primary/10",
      warning: "border-accent-warning/20 bg-accent-warning/10",
      success: "border-accent-success/20 bg-accent-success/10",
      danger: "border-accent-danger/20 bg-accent-danger/10",
      neutral: "border-border-default/70 bg-surface-overlay/40",
    },
  },
  defaultVariants: {
    tone: "neutral",
  },
});

const noticeTitleVariants = cva("text-sm font-medium", {
  variants: {
    tone: {
      info: "text-accent-primary",
      warning: "text-accent-warning",
      success: "text-accent-success",
      danger: "text-accent-danger",
      neutral: "text-text-primary",
    },
  },
  defaultVariants: {
    tone: "neutral",
  },
});

export interface NoticeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof noticeVariants> {}

export function Notice({ className, tone, ...props }: NoticeProps) {
  return <div className={cn(noticeVariants({ tone, className }))} {...props} />;
}

export interface NoticeTitleProps
  extends React.HTMLAttributes<HTMLParagraphElement>,
    VariantProps<typeof noticeTitleVariants> {}

export function NoticeTitle({ className, tone, ...props }: NoticeTitleProps) {
  return <p className={cn(noticeTitleVariants({ tone, className }))} {...props} />;
}
