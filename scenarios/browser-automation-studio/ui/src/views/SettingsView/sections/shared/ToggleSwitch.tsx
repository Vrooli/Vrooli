interface ToggleSwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
  icon?: React.ReactNode;
  className?: string;
}

export function ToggleSwitch({ checked, onChange, label, description, icon, className = '' }: ToggleSwitchProps) {
  return (
    <div className={`flex items-center justify-between py-2 ${className}`}>
      <div className="flex items-center gap-3">
        {icon && <span className="text-gray-400">{icon}</span>}
        <div>
          <label className="text-sm text-gray-300 block">{label}</label>
          {description && <span className="text-xs text-gray-500">{description}</span>}
        </div>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
          checked ? 'bg-flow-accent' : 'bg-gray-700'
        }`}
      >
        <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${checked ? 'translate-x-6' : 'translate-x-1'}`} />
      </button>
    </div>
  );
}
