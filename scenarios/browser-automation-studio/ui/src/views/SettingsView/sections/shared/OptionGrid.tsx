interface OptionGridOption<T extends string> {
  id: T;
  label: string;
  subtitle?: string;
  description?: string;
  preview?: React.ReactNode;
  previewStyle?: React.CSSProperties;
  previewNode?: React.ReactNode;
}

interface OptionGridProps<T extends string> {
  options: OptionGridOption<T>[];
  value: T;
  onChange: (value: T) => void;
  columns?: 2 | 3 | 4;
}

export function OptionGrid<T extends string>({ options, value, onChange, columns = 3 }: OptionGridProps<T>) {
  const gridCols = columns === 2 ? 'grid-cols-2' : columns === 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-2 sm:grid-cols-3';

  return (
    <div className={`grid ${gridCols} gap-2`}>
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          onClick={() => onChange(option.id)}
          className={`flex flex-col items-center p-3 rounded-lg border transition-all text-center ${
            value === option.id
              ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
              : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
          }`}
        >
          {option.preview && <div className="mb-2">{option.preview}</div>}
          {option.previewNode && <div className="relative w-10 h-10 mb-2 rounded-lg overflow-hidden" style={option.previewStyle}>{option.previewNode}</div>}
          {!option.preview && !option.previewNode && option.previewStyle && (
            <div className="w-10 h-10 mb-2 rounded-lg" style={option.previewStyle} />
          )}
          <span className="text-sm font-medium text-surface">{option.label}</span>
          {(option.subtitle || option.description) && (
            <span className="text-xs text-gray-500 mt-0.5">{option.subtitle || option.description}</span>
          )}
        </button>
      ))}
    </div>
  );
}
