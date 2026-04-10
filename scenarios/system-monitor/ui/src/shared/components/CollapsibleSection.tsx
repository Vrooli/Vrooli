import { useState } from 'react';
import type { ReactNode } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { SearchInput } from './SearchInput';

interface CollapsibleSectionProps {
  title: string;
  /** Text shown in parentheses next to title */
  count?: string | number;
  /** Whether the section starts expanded */
  defaultExpanded?: boolean;
  /** Optional search input config */
  search?: {
    placeholder: string;
    value: string;
    onChange: (value: string) => void;
  };
  children: ReactNode;
}

export const CollapsibleSection = ({
  title,
  count,
  defaultExpanded = true,
  search,
  children
}: CollapsibleSectionProps) => {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div className="collapsible-section">
      <div
        className="section-header hover-bg-accent"
        style={{
          borderBottom: expanded ? '1px solid var(--color-primary)' : 'none'
        }}
      >
        <div
          className="section-header-toggle"
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
          <h3 className="section-heading text-md">
            {title}
          </h3>
          {count != null && (
            <span className="text-muted text-sm" style={{ marginLeft: 'var(--spacing-xs)' }}>
              ({count})
            </span>
          )}
        </div>

        {expanded && search && (
          <SearchInput
            placeholder={search.placeholder}
            value={search.value}
            onChange={search.onChange}
          />
        )}
      </div>

      {expanded && children}
    </div>
  );
};
