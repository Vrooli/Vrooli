import { CheckCircle } from 'lucide-react';
import { type Template } from '../lib/api';

interface TemplateCardProps {
  template: Template;
  isSelected: boolean;
  onSelect: (id: string) => void;
}

function getCardClassName(isSelected: boolean): string {
  const base = 'rounded-xl border p-4 sm:p-5 text-left transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950';

  if (isSelected) {
    return `${base} border-emerald-400/60 bg-emerald-500/5 shadow-lg shadow-emerald-500/10 scale-[1.02]`;
  }

  return `${base} border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10 hover:scale-[1.01]`;
}

function getAriaLabel(template: Template, isSelected: boolean): string {
  const baseLabel = `Select ${template.name} template, version ${template.version}`;
  return isSelected ? `${baseLabel} (currently selected)` : baseLabel;
}

export function TemplateCard({ template, isSelected, onSelect }: TemplateCardProps) {
  return (
    <button
      key={template.id}
      role="listitem"
      data-testid={`template-card-${template.id}`}
      onClick={() => onSelect(template.id)}
      className={getCardClassName(isSelected)}
      aria-current={isSelected ? 'true' : 'false'}
      aria-label={getAriaLabel(template, isSelected)}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="text-xs font-medium text-slate-400 uppercase tracking-wide mb-1">Template</div>
          <div className="text-base sm:text-lg font-semibold truncate" data-testid={`template-name-${template.id}`}>
            {template.name}
          </div>
        </div>
        <span
          className="flex-shrink-0 text-xs text-slate-400 border border-white/10 rounded-full px-2 py-1"
          data-testid={`template-version-${template.id}`}
        >
          v{template.version}
        </span>
      </div>
      <p className="text-xs sm:text-sm text-slate-300 mt-2 line-clamp-2 leading-relaxed">{template.description}</p>
      {isSelected && (
        <div className="mt-3 pt-3 border-t border-emerald-500/20">
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-300">
            <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
            Selected
          </span>
        </div>
      )}
    </button>
  );
}
