export const StatCard = ({ label, value, description }: { label: string; value: string; description?: string }) => (
  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
    <p className="text-xs uppercase tracking-[0.2em] text-white/60">{label}</p>
    <p className="mt-2 text-3xl font-semibold">{value}</p>
    {description ? <p className="text-sm text-white/60">{description}</p> : null}
  </div>
);
