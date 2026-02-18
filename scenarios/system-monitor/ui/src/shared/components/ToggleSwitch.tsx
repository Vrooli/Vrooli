interface ToggleSwitchProps {
  checked: boolean;
  onChange: () => void;
  disabled?: boolean;
  title?: string;
  size?: 'sm' | 'md';
}

export const ToggleSwitch = ({ checked, onChange, disabled, title, size = 'md' }: ToggleSwitchProps) => {
  const classes = [
    'toggle-switch',
    checked && 'toggle-switch-on',
    size === 'sm' && 'toggle-switch-sm',
    disabled && 'toggle-switch-disabled',
  ].filter(Boolean).join(' ');

  return (
    <button
      className={classes}
      onClick={disabled ? undefined : onChange}
      role="switch"
      aria-checked={checked}
      aria-disabled={disabled || undefined}
      title={title}
    />
  );
};
