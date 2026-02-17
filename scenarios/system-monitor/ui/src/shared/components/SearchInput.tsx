import type { ReactNode } from 'react';
import { Search } from 'lucide-react';

interface SearchInputProps {
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
  icon?: ReactNode;
}

export const SearchInput = ({ placeholder, value, onChange, icon }: SearchInputProps) => (
  <div className="search-input-wrapper">
    <span className="search-input-icon">
      {icon ?? <Search size={14} />}
    </span>
    <input
      type="text"
      placeholder={placeholder}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onClick={(e) => e.stopPropagation()}
      className="search-input"
    />
  </div>
);
