/**
 * Generic toggle button group for settings forms.
 * Renders a row of mutually-exclusive options styled as bordered buttons.
 */

export function ToggleButtons<T extends string | boolean>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
}) {
  return (
    <div className={`mt-2 grid gap-2`} style={{ gridTemplateColumns: `repeat(${options.length}, 1fr)` }}>
      {options.map((opt) => (
        <button
          key={String(opt.value)}
          className={`rounded-lg border py-2 text-sm font-medium ${
            value === opt.value
              ? "border-cyan-500 bg-slate-900 text-cyan-400"
              : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"
          }`}
          onClick={() => onChange(opt.value)}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
