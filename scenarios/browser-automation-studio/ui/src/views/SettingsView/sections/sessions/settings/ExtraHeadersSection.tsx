import { Plus, Trash2 } from 'lucide-react';
import { useState, useMemo } from 'react';
import { validateHttpHeader, isBlockedHeader, FormFieldGroup, FORM_INPUT_CLASSES } from '@/components/form';

/** Input classes with error state */
const ERROR_INPUT_CLASSES =
  'flex-1 rounded-md border border-red-300 dark:border-red-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm font-mono text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-red-500 focus:border-transparent focus:outline-none';

/** Input classes for header name (monospace font) */
const HEADER_NAME_INPUT_CLASSES = `${FORM_INPUT_CLASSES} flex-1 font-mono`;

/** Input classes for header value (flex-2 for wider) */
const HEADER_VALUE_INPUT_CLASSES = `${FORM_INPUT_CLASSES} flex-[2]`;

interface ExtraHeadersSectionProps {
  headers: Record<string, string>;
  onAdd: (key: string, value: string) => void;
  onUpdate: (oldKey: string, newKey: string, value: string) => void;
  onRemove: (key: string) => void;
}

export function ExtraHeadersSection({ headers, onAdd, onUpdate, onRemove }: ExtraHeadersSectionProps) {
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  const [error, setError] = useState<string | null>(null);

  const headerEntries = Object.entries(headers);

  // Validate the new header name as user types
  const newKeyValidation = useMemo(() => {
    if (!newKey.trim()) return undefined;
    return validateHttpHeader(newKey.trim());
  }, [newKey]);

  const handleAdd = () => {
    const trimmedKey = newKey.trim();
    const trimmedValue = newValue.trim();

    // Use the validation utility
    const validationError = validateHttpHeader(trimmedKey);
    if (validationError) {
      setError(validationError);
      return;
    }

    if (headers[trimmedKey] !== undefined) {
      setError(`Header "${trimmedKey}" already exists`);
      return;
    }

    onAdd(trimmedKey, trimmedValue);
    setNewKey('');
    setNewValue('');
    setError(null);
  };

  const handleKeyChange = (oldKey: string, newKey: string, value: string) => {
    const trimmedKey = newKey.trim();

    // Reject blocked headers during edit
    if (isBlockedHeader(trimmedKey)) {
      return;
    }

    onUpdate(oldKey, trimmedKey, value);
  };

  return (
    <div className="space-y-6">
      {/* Description */}
      <p className="text-sm text-gray-600 dark:text-gray-400">
        Custom HTTP headers are sent with every request in this session. Useful for authentication tokens, request tracing, or API keys.
      </p>

      {/* Existing Headers */}
      {headerEntries.length > 0 && (
        <FormFieldGroup title="Current Headers">
          <div className="space-y-2">
            {headerEntries.map(([key, value]) => (
              <div key={key} className="flex items-center gap-2">
                <input
                  type="text"
                  value={key}
                  onChange={(e) => handleKeyChange(key, e.target.value, value)}
                  placeholder="Header-Name"
                  className={HEADER_NAME_INPUT_CLASSES}
                />
                <input
                  type="text"
                  value={value}
                  onChange={(e) => onUpdate(key, key, e.target.value)}
                  placeholder="Header value"
                  className={HEADER_VALUE_INPUT_CLASSES}
                />
                <button
                  type="button"
                  onClick={() => onRemove(key)}
                  className="p-2 text-gray-400 hover:text-red-500 dark:hover:text-red-400 transition-colors"
                  title="Remove header"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            ))}
          </div>
        </FormFieldGroup>
      )}

      {/* Add New Header */}
      <FormFieldGroup
        title="Add Header"
        className="pt-4 border-t border-gray-200 dark:border-gray-700"
      >
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={newKey}
            onChange={(e) => {
              setNewKey(e.target.value);
              setError(null);
            }}
            placeholder="X-Custom-Header"
            className={newKeyValidation ? ERROR_INPUT_CLASSES : HEADER_NAME_INPUT_CLASSES}
            aria-invalid={!!newKeyValidation}
          />
          <input
            type="text"
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            placeholder="Header value"
            className={HEADER_VALUE_INPUT_CLASSES}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                handleAdd();
              }
            }}
          />
          <button
            type="button"
            onClick={handleAdd}
            disabled={!!newKeyValidation}
            className="p-2 text-gray-600 hover:text-blue-500 dark:text-gray-400 dark:hover:text-blue-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Add header"
          >
            <Plus size={16} />
          </button>
        </div>
        {(error || newKeyValidation) && (
          <p className="text-xs text-red-500 dark:text-red-400 mt-1">{error || newKeyValidation}</p>
        )}
      </FormFieldGroup>

      {/* Info Box */}
      <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
        <p className="text-xs text-amber-700 dark:text-amber-300">
          <strong>Note:</strong> Headers like <code className="bg-amber-100 dark:bg-amber-900/50 px-1 rounded">Host</code>,{' '}
          <code className="bg-amber-100 dark:bg-amber-900/50 px-1 rounded">Content-Length</code>, and{' '}
          <code className="bg-amber-100 dark:bg-amber-900/50 px-1 rounded">Cookie</code> cannot be set here.
          Use the Storage State section for cookies.
        </p>
      </div>

      {/* Empty State */}
      {headerEntries.length === 0 && (
        <div className="text-center py-6 text-gray-500 dark:text-gray-400">
          <p className="text-sm">No custom headers configured</p>
          <p className="text-xs mt-1">Add headers above to include them in every request</p>
        </div>
      )}
    </div>
  );
}
