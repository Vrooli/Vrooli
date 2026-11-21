interface DownloadStorefrontFieldsProps {
  title: string;
  enabled: boolean;
  onEnabledChange: (value: boolean) => void;
  labelValue: string;
  urlValue: string;
  badgeValue: string;
  onLabelChange: (value: string) => void;
  onUrlChange: (value: string) => void;
  onBadgeChange: (value: string) => void;
}

export function DownloadStorefrontFields({
  title,
  enabled,
  onEnabledChange,
  labelValue,
  urlValue,
  badgeValue,
  onLabelChange,
  onUrlChange,
  onBadgeChange,
}: DownloadStorefrontFieldsProps) {
  return (
    <div
      className={`space-y-3 rounded-2xl border p-4 transition-opacity ${
        enabled ? 'border-white/10 bg-surface-darker' : 'border-white/5 bg-surface-darker/50 opacity-60'
      }`}
    >
      <div className="flex items-center justify-between">
        <p className="text-sm font-semibold text-white">{title}</p>
        <label className="flex items-center gap-2 text-xs text-slate-400">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(event) => { onEnabledChange(event.target.checked); }}
            className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
          />
          Enabled
        </label>
      </div>
      <DownloadStorefrontInput label="Label" value={labelValue} enabled={enabled} onChange={onLabelChange} />
      <DownloadStorefrontInput label="URL" value={urlValue} enabled={enabled} onChange={onUrlChange} />
      <DownloadStorefrontInput label="Badge text (optional)" value={badgeValue} enabled={enabled} onChange={onBadgeChange} />
    </div>
  );
}

interface DownloadStorefrontInputProps {
  label: string;
  value: string;
  enabled: boolean;
  onChange: (value: string) => void;
}

function DownloadStorefrontInput({ label, value, enabled, onChange }: DownloadStorefrontInputProps) {
  return (
    <div className="space-y-2">
      <label className="text-xs text-slate-500">{label}</label>
      <input
        type="text"
        value={value}
        disabled={!enabled}
        onChange={(event) => { onChange(event.target.value); }}
        className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
      />
    </div>
  );
}
