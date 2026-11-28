import { AlertTriangle } from "lucide-react";
import type { DiscoveredCertificate } from "../../domain/signing";

interface DiscoveredCertSelectorProps {
  label: string;
  discovered: DiscoveredCertificate[];
  onSelect: (cert: DiscoveredCertificate) => void;
  expiryWarningText?: string;
}

export function DiscoveredCertSelector({
  label,
  discovered,
  onSelect,
  expiryWarningText = "Some items expire within 30 days.",
}: DiscoveredCertSelectorProps) {
  if (discovered.length === 0) return null;

  const hasExpiringSoon = discovered.some(
    (c) => (c.days_to_expiry ?? Infinity) <= 30 && !c.is_expired,
  );

  return (
    <div className="rounded border border-slate-800 bg-slate-950/50 p-2 text-xs text-slate-200 space-y-2">
      <div className="flex items-center gap-2">
        <span className="font-medium">{label}:</span>
        <select
          aria-label={label}
          className="flex-1 rounded border border-slate-700 bg-slate-900 px-2 py-1"
          onChange={(e) => {
            const selected = discovered.find((c) => c.id === e.target.value);
            if (selected) {
              onSelect(selected);
            }
          }}
          defaultValue=""
        >
          <option value="">Select to apply</option>
          {discovered.map((cert) => (
            <option key={cert.id} value={cert.id}>
              {cert.name || cert.subject || cert.id}
            </option>
          ))}
        </select>
      </div>
      {hasExpiringSoon && (
        <div className="flex items-center gap-1 text-amber-300">
          <AlertTriangle className="h-3 w-3" />
          <span>{expiryWarningText}</span>
        </div>
      )}
    </div>
  );
}
