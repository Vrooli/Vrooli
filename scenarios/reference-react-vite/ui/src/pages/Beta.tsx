export function Beta() {
  return (
    <div data-testid="beta-page" className="flex flex-col gap-2">
      <h2 className="text-2xl font-semibold">Beta features</h2>
      <p className="text-sm text-slate-400">Visible only when the feature_beta flag is on.</p>
    </div>
  );
}
