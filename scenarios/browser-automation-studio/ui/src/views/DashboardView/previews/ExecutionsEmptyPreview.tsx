import React from 'react';
import { Activity, CheckCircle2, Clock, Eye, MonitorPlay } from 'lucide-react';

export const ExecutionsEmptyPreview: React.FC = () => {
  const steps = [
    { label: 'Navigate to dashboard', status: 'done' },
    { label: 'Click login & submit', status: 'done' },
    { label: 'Verify success banner', status: 'active' },
    { label: 'Capture screenshot', status: 'queued' },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between rounded-xl border border-green-500/30 bg-green-500/10 px-4 py-3 shadow-md shadow-green-500/10 animate-fade-in-up">
        <div className="flex items-center gap-2 text-green-200">
          <MonitorPlay size={18} />
          <span className="font-semibold text-surface">Live execution preview</span>
        </div>
        <div className="flex items-center gap-1 text-xs text-green-200 bg-green-500/20 px-2 py-1 rounded-lg">
          <Activity size={12} className="animate-pulse" />
          Running
        </div>
      </div>

      <div className="space-y-2">
        {steps.map((step, idx) => {
          const statusStyles = {
            done: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200',
            active: 'border-flow-accent/40 bg-flow-accent/10 text-surface',
            queued: 'border-amber-500/30 bg-amber-500/10 text-amber-200',
          } as const;

          return (
            <div
              key={step.label}
              className={`flex items-center justify-between rounded-lg border px-3 py-2 ${statusStyles[step.status as keyof typeof statusStyles]}`}
              style={{ animation: 'fade-in-up 0.4s ease-out both', animationDelay: `${idx * 60}ms` }}
            >
              <div className="flex items-center gap-2 text-sm">
                {step.status === 'done' && <CheckCircle2 size={16} />}
                {step.status === 'active' && <Eye size={16} className="animate-pulse" />}
                {step.status === 'queued' && <Clock size={16} />}
                <span>{step.label}</span>
              </div>
              <div className="text-xs opacity-80">
                {step.status === 'done' && 'Verified'}
                {step.status === 'active' && 'Streaming'}
                {step.status === 'queued' && 'Next'}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default ExecutionsEmptyPreview;
