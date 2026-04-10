import type { LucideIcon } from "lucide-react";

export function SectionTitle({ icon: Icon, children }: { icon: LucideIcon; children: React.ReactNode }) {
  return (
    <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-100">
      <Icon className="h-4 w-4 text-slate-400" />
      {children}
    </h3>
  );
}

export function ActionRow({
  icon: Icon,
  title,
  subtitle,
  children,
}: {
  icon: LucideIcon;
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-slate-900/50 px-3 py-2.5">
      <div className="flex items-center gap-2.5 min-w-0">
        <Icon className="h-4 w-4 shrink-0 text-slate-400" />
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-200">{title}</p>
          <p className="text-xs text-slate-500">{subtitle}</p>
        </div>
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

export function InfoItem({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="space-y-0.5">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="text-sm text-slate-200">{value || "—"}</p>
    </div>
  );
}
