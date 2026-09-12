import type { BootRecord } from '../types';

interface BootPickerProps {
  boots: BootRecord[];
  value: string;
  onChange: (boot: string) => void;
}

export const BootPicker = ({ boots, value, onChange }: BootPickerProps) => (
  <label
    className="text-xs text-muted"
    data-sm-style="sm-style-975684b32c"
  >
    Boot
    <select value={value} onChange={(e) => { onChange(e.target.value); }}>
      <option value="">All boots</option>
      {boots.map((b) => (
        <option key={b.bootId || `idx-${b.index}`} value={b.bootId || String(b.index)}>
          {b.index === 0 ? 'current' : b.index} · {b.bootId.slice(0, 8) || '—'}
        </option>
      ))}
    </select>
  </label>
);
