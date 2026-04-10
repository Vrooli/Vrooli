import { FileText } from 'lucide-react';
import type { InvestigationScript } from '../../../types';

interface ScriptListItemProps {
  script: InvestigationScript;
  isSelected: boolean;
  onSelect: (script: InvestigationScript) => void;
}

export const ScriptListItem = ({ script, isSelected, onSelect }: ScriptListItemProps) => (
  <button
    type="button"
    onClick={() => onSelect(script)}
    style={{
      width: '100%',
      textAlign: 'left',
      padding: 'var(--spacing-md)',
      border: 'none',
      background: isSelected ? 'var(--color-primary-muted)' : 'transparent',
      color: 'var(--color-text)',
      display: 'flex',
      flexDirection: 'column',
      gap: 'var(--spacing-xs)',
      cursor: 'pointer'
    }}
  >
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
      <span className="icon-text" style={{
        fontSize: 'var(--text-sm)',
        fontWeight: 600,
        textTransform: 'uppercase'
      }}>
        <FileText size={16} style={{ color: 'var(--color-primary)' }} />
        {script.name}
      </span>
      <span style={{
        fontSize: 'var(--text-xs)',
        color: script.enabled ? 'var(--color-success)' : 'var(--color-text-secondary)',
        letterSpacing: '0.08em'
      }}>
        {script.enabled ? 'ENABLED' : 'DISABLED'}
      </span>
    </div>
    <div className="text-dim-xs" style={{
      display: 'flex',
      justifyContent: 'space-between',
    }}>
      <span>{script.category}</span>
      <span>{script.author}</span>
    </div>
    <p style={{
      margin: 0,
      fontSize: 'var(--text-xs)',
      color: 'var(--color-text-secondary)'
    }}>
      {script.description}
    </p>
  </button>
);
