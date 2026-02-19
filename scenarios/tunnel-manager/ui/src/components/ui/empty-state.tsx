import type { ReactNode, ComponentType } from "react";

interface EmptyStateProps {
  icon?: ComponentType<{ className?: string }>;
  title: string;
  description?: string;
  children?: ReactNode;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, children, className = "" }: EmptyStateProps) {
  return (
    <div className={`rounded-lg border border-dashed border-white/10 p-6 text-center ${className}`}>
      {Icon && <Icon className="mx-auto h-8 w-8 text-slate-600" />}
      <p className={`${Icon ? "mt-2" : ""} text-sm font-medium text-slate-300`}>{title}</p>
      {description && <p className="mt-1 text-xs text-slate-500">{description}</p>}
      {children}
    </div>
  );
}
