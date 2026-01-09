import { useState } from 'react';
import { Copy, ShieldCheck } from 'lucide-react';
import { ADMIN_CREDENTIALS_NOTE, DEFAULT_ADMIN_EMAIL, DEFAULT_ADMIN_PASSWORD } from '../consts/credentials';

interface AdminCredentialsHintProps {
  className?: string;
  compact?: boolean;
}

export function AdminCredentialsHint({ className = '', compact = false }: AdminCredentialsHintProps) {
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const handleCopy = async (label: string, value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(label);
      setTimeout(() => setCopiedField(null), 1500);
    } catch (error) {
      console.warn('Failed to copy credentials to clipboard:', error);
    }
  };

  return (
    <div
      className={`rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100 shadow-emerald-500/10 ${className}`}
      data-testid="admin-credentials-hint"
    >
      <div className="flex items-center gap-2 text-emerald-200 font-semibold text-xs uppercase tracking-wide mb-2">
        <ShieldCheck className="h-4 w-4" aria-hidden="true" />
        Default Admin Credentials
      </div>
      <div className={`flex ${compact ? 'flex-col gap-2' : 'flex-wrap gap-3 items-center'}`}>
        <CredentialBadge
          label="Email"
          value={DEFAULT_ADMIN_EMAIL}
          onCopy={() => handleCopy('email', DEFAULT_ADMIN_EMAIL)}
          copied={copiedField === 'email'}
        />
        <CredentialBadge
          label="Password"
          value={DEFAULT_ADMIN_PASSWORD}
          onCopy={() => handleCopy('password', DEFAULT_ADMIN_PASSWORD)}
          copied={copiedField === 'password'}
        />
      </div>
      <p className="text-xs text-emerald-200/70 mt-2 leading-relaxed">
        {ADMIN_CREDENTIALS_NOTE}
      </p>
    </div>
  );
}

interface CredentialBadgeProps {
  label: string;
  value: string;
  copied: boolean;
  onCopy: () => void;
}

function CredentialBadge({ label, value, copied, onCopy }: CredentialBadgeProps) {
  return (
    <div className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5">
      <span className="text-xs uppercase tracking-wide text-emerald-200/80">{label}</span>
      <code className="text-sm text-white font-mono">{value}</code>
      <button
        type="button"
        onClick={onCopy}
        className="inline-flex items-center gap-1 text-xs text-emerald-200/80 hover:text-white transition-colors focus:outline-none"
        aria-label={`Copy ${label}`}
      >
        <Copy className={`h-3.5 w-3.5 ${copied ? 'text-emerald-300' : ''}`} aria-hidden="true" />
        <span className="hidden sm:inline">{copied ? 'Copied' : 'Copy'}</span>
      </button>
    </div>
  );
}
