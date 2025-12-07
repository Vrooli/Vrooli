import React from 'react';
import { FolderOpen, Layout, Layers, ShieldCheck, Sparkles } from 'lucide-react';

export const ProjectsEmptyPreview: React.FC = () => {
  const workflows = [
    { name: 'Login & Smoke Test', status: 'Ready', color: 'text-blue-300' },
    { name: 'Pricing Page Scraper', status: 'Updated 2h ago', color: 'text-emerald-300' },
    { name: 'Weekly Report Export', status: 'Queued', color: 'text-amber-300' },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-flow-accent/25 to-purple-500/30 text-white shadow-md shadow-blue-500/20 animate-fade-in-up">
          <FolderOpen size={22} />
        </div>
        <div>
          <div className="text-sm uppercase tracking-wide text-flow-text-muted">Portfolio</div>
          <div className="text-lg font-semibold text-white">Growth Experiments</div>
        </div>
      </div>

      <div className="rounded-xl border border-flow-border/60 bg-flow-surface/70 p-4 shadow-inner shadow-blue-500/10 animate-fade-in-up">
        <div className="flex items-center justify-between text-xs text-flow-text-secondary mb-3">
          <span className="flex items-center gap-2">
            <Layers size={14} />
            Workflows
          </span>
          <span className="flex items-center gap-1 text-flow-accent">
            <Sparkles size={14} />
            AI ready
          </span>
        </div>
        <div className="space-y-2">
          {workflows.map((workflow, idx) => (
            <div
              key={workflow.name}
              className="flex items-center justify-between rounded-lg border border-flow-border/40 bg-flow-node/80 px-3 py-2"
              style={{ animation: 'fade-in-up 0.4s ease-out both', animationDelay: `${idx * 70}ms` }}
            >
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-flow-accent animate-pulse-slow" />
                <span className="text-white text-sm">{workflow.name}</span>
              </div>
              <span className={`text-xs ${workflow.color}`}>{workflow.status}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 text-xs text-flow-text-secondary">
        <div className="flex items-center gap-2 rounded-lg border border-flow-border/60 bg-flow-node/60 px-3 py-2">
          <ShieldCheck size={14} className="text-emerald-400" />
          Versioned history
        </div>
        <div className="flex items-center gap-2 rounded-lg border border-flow-border/60 bg-flow-node/60 px-3 py-2">
          <Layout size={14} className="text-blue-300" />
          Organized folders
        </div>
      </div>
    </div>
  );
};

export default ProjectsEmptyPreview;
