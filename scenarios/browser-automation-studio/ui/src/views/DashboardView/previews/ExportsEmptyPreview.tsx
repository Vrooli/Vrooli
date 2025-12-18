import React from 'react';
import { ArrowDownToLine, FileVideo2, FileJson2, Image, Sparkles } from 'lucide-react';

export const ExportsEmptyPreview: React.FC = () => {
  const formats = [
    { icon: <FileVideo2 size={18} />, label: 'MP4 replay', accent: 'text-blue-200 bg-blue-500/10 border-blue-500/30' },
    { icon: <Image size={18} />, label: 'GIF highlights', accent: 'text-amber-200 bg-amber-500/10 border-amber-500/30' },
    { icon: <FileJson2 size={18} />, label: 'JSON package', accent: 'text-emerald-200 bg-emerald-500/10 border-emerald-500/30' },
  ];

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-flow-border/60 bg-flow-node/70 px-4 py-3 flex items-center justify-between shadow-sm shadow-black/20 animate-fade-in-up">
        <div className="flex items-center gap-2 text-purple-100">
          <Sparkles size={18} />
          <span className="font-semibold text-surface">Choose your export</span>
        </div>
        <div className="flex items-center gap-2 text-xs text-flow-text-secondary">
          <ArrowDownToLine size={12} className="text-flow-text-muted" />
          <span className="text-flow-text-secondary">Ready</span>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
        {formats.map((format, idx) => (
          <div
            key={format.label}
            className={`flex items-center gap-2 rounded-lg border px-3 py-2 ${format.accent}`}
            style={{ animation: 'fade-in-up 0.4s ease-out both', animationDelay: `${idx * 70}ms` }}
          >
            {format.icon}
            <span className="text-sm font-medium text-surface">{format.label}</span>
          </div>
        ))}
      </div>

      <div className="rounded-lg border border-flow-border/60 bg-flow-node/70 p-3 text-xs text-flow-text-secondary leading-relaxed animate-fade-in-up">
        Preview how exports capture logs, screenshots, and data snapshots so you can drop them into docs, tickets, or CI reports instantly.
      </div>
    </div>
  );
};

export default ExportsEmptyPreview;
