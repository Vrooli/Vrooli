import type { CSSProperties, ReactNode } from "react";
import { cn } from "../../lib/utils";
import { Button, type ButtonProps } from "./button";

interface FloatingActionButtonProps extends Omit<ButtonProps, "children"> {
  icon: ReactNode;
  label: string;
}

export function FloatingActionButton({
  icon,
  label,
  className,
  style,
  type = "button",
  ...props
}: FloatingActionButtonProps) {
  const mergedStyle: CSSProperties = {
    bottom: "calc(5rem + env(safe-area-inset-bottom))",
    ...style,
  };

  return (
    <Button
      type={type}
      aria-label={label}
      className={cn(
        "fixed right-4 z-30 h-12 w-12 p-0 shadow-lg",
        "bg-cyan-500 text-slate-950 hover:bg-cyan-400",
        className
      )}
      style={mergedStyle}
      {...props}
    >
      {icon}
    </Button>
  );
}
