import { ReactNode } from 'react';
import { Info, AlertTriangle, XCircle, CheckCircle2, Lightbulb, type LucideIcon } from 'lucide-react';
import { cn } from '../../../shared/lib/utils';

export type CalloutType = 'info' | 'warning' | 'error' | 'success' | 'tip';

interface CalloutAction {
  label: string;
  onClick: () => void;
}

interface CalloutProps {
  type: CalloutType;
  title?: string;
  message: ReactNode;
  icon?: LucideIcon;
  actions?: CalloutAction[];
  className?: string;
}

const CALLOUT_CONFIG: Record<CalloutType, { border: string; bg: string; text: string; DefaultIcon: LucideIcon }> = {
  info: {
    border: 'border-blue-500/20',
    bg: 'bg-blue-500/10',
    text: 'text-blue-300',
    DefaultIcon: Info,
  },
  warning: {
    border: 'border-amber-500/20',
    bg: 'bg-amber-500/10',
    text: 'text-amber-300',
    DefaultIcon: AlertTriangle,
  },
  error: {
    border: 'border-rose-500/20',
    bg: 'bg-rose-500/10',
    text: 'text-rose-300',
    DefaultIcon: XCircle,
  },
  success: {
    border: 'border-emerald-500/20',
    bg: 'bg-emerald-500/10',
    text: 'text-emerald-300',
    DefaultIcon: CheckCircle2,
  },
  tip: {
    border: 'border-purple-500/20',
    bg: 'bg-purple-500/10',
    text: 'text-purple-300',
    DefaultIcon: Lightbulb,
  },
};

export function Callout({ type, title, message, icon, actions, className }: CalloutProps) {
  const config = CALLOUT_CONFIG[type];
  const Icon = icon ?? config.DefaultIcon;

  return (
    <div
      className={cn(
        'flex items-start gap-3 rounded-xl border p-4',
        config.border,
        config.bg,
        className
      )}
    >
      <Icon className={cn('h-5 w-5 flex-shrink-0 mt-0.5', config.text)} />
      <div className="flex-1 min-w-0">
        {title && (
          <p className={cn('text-sm font-medium', config.text)}>{title}</p>
        )}
        <div className={cn('text-sm', title ? 'text-slate-300 mt-1' : config.text)}>
          {message}
        </div>
        {actions && actions.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-3">
            {actions.map((action, index) => (
              <button
                key={index}
                onClick={action.onClick}
                className={cn(
                  'text-xs font-medium px-3 py-1.5 rounded-lg transition-colors',
                  'bg-white/5 hover:bg-white/10',
                  config.text
                )}
              >
                {action.label}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
