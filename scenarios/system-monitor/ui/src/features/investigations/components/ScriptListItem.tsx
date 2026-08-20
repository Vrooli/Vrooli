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
    onClick={() => { onSelect(script); }}
    className={`script-list-item ${isSelected ? 'is-selected' : ''}`}
  >
    <div data-sm-style="sm-style-d4e3683f04">
      <span className="icon-text" data-sm-style="sm-style-006c1c4c18">
        <FileText size={16} data-sm-style="sm-style-392c7463c7" />
        {script.name}
      </span>
      <span className={`text-xs script-enabled-label ${script.enabled ? 'is-enabled' : 'is-disabled'}`}>
        {script.enabled ? 'ENABLED' : 'DISABLED'}
      </span>
    </div>
    <div data-sm-style="sm-style-b34553c6d3">
      <span className="text-dim-xs" title={script.executionMode === 'native' ? 'Runs through typed collector queries' : 'Requires declared host tools'}>
        {script.executionMode === 'native' ? 'NATIVE QUERY' : 'SHELL-GATED'}
      </span>
      {script.skipReason && <span className="text-dim-xs" title={script.skipReason}>UNAVAILABLE ON HOST</span>}
    </div>
    <div className="text-dim-xs" data-sm-style="sm-style-a37b6e5cc8">
      <span>{script.category}</span>
      <span>{script.author}</span>
    </div>
    <p data-sm-style="sm-style-4d1bf7fe88">
      {script.description}
    </p>
  </button>
);
