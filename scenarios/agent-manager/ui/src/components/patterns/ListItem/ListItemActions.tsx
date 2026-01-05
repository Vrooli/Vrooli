import * as React from "react";
import { cn } from "../../../lib/utils";
import { Button } from "../../ui/button";

interface ListItemActionsProps {
  children: React.ReactNode;
  className?: string;
}

export function ListItemActions({ children, className }: ListItemActionsProps) {
  return <div className={cn("flex items-center gap-1", className)}>{children}</div>;
}

interface ListItemActionButtonProps {
  icon: React.ReactNode;
  onClick: (e: React.MouseEvent) => void;
  label: string;
  variant?: "ghost" | "destructive";
  className?: string;
}

export function ListItemActionButton({
  icon,
  onClick,
  label,
  variant = "ghost",
  className,
}: ListItemActionButtonProps) {
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onClick(e);
  };

  return (
    <Button
      variant={variant === "destructive" ? "ghost" : "ghost"}
      size="icon"
      className={cn(
        "h-6 w-6",
        variant === "destructive" && "text-destructive hover:text-destructive",
        className
      )}
      aria-label={label}
      onClick={handleClick}
    >
      {icon}
    </Button>
  );
}
