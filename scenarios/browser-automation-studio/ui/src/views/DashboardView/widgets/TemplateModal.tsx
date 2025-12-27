import { useState, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { X, Play, Globe } from 'lucide-react';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { BrowserUrlBar } from '@/domains/recording';
import type { Template, TemplateField } from './TemplatesGallery';

interface TemplateModalProps {
  isOpen: boolean;
  onClose: () => void;
  template: Template | null;
}

/**
 * Build a prompt by replacing placeholders with values.
 * Placeholders are in the format {{key}}.
 * For optional fields with empty values, the placeholder is removed.
 */
function buildPrompt(
  promptTemplate: string,
  url: string,
  fieldValues: Record<string, string>,
  fields: TemplateField[]
): string {
  let result = promptTemplate;

  // Replace {{url}} placeholder
  result = result.replace(/\{\{url\}\}/g, url);

  // Replace field placeholders
  for (const field of fields) {
    const value = fieldValues[field.key] || field.defaultValue || '';
    const placeholder = new RegExp(`\\{\\{${field.key}\\}\\}`, 'g');

    if (value) {
      // For optional fields that append text, add a space before if value exists
      if (!field.required && field.defaultValue === '') {
        result = result.replace(placeholder, ` ${value}`);
      } else {
        result = result.replace(placeholder, value);
      }
    } else {
      // Remove placeholder for empty optional fields
      result = result.replace(placeholder, '');
    }
  }

  // Clean up any double spaces
  result = result.replace(/\s+/g, ' ').trim();

  return result;
}

export function TemplateModal({ isOpen, onClose, template }: TemplateModalProps) {
  const navigate = useNavigate();
  const [url, setUrl] = useState('');
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({});

  // Reset state when template changes
  const handleClose = useCallback(() => {
    setUrl('');
    setFieldValues({});
    onClose();
  }, [onClose]);

  // Check if all required fields are filled and get validation message
  const { isValid, validationMessage } = useMemo(() => {
    if (!template) return { isValid: false, validationMessage: '' };
    if (!url.trim()) return { isValid: false, validationMessage: 'Enter a website URL to continue' };

    const missingFields: string[] = [];
    for (const field of template.fields) {
      if (field.required && !fieldValues[field.key]?.trim()) {
        missingFields.push(field.label);
      }
    }

    if (missingFields.length > 0) {
      const fieldList = missingFields.length === 1
        ? missingFields[0]
        : `${missingFields.slice(0, -1).join(', ')} and ${missingFields[missingFields.length - 1]}`;
      return { isValid: false, validationMessage: `Fill in ${fieldList} to continue` };
    }

    return { isValid: true, validationMessage: '' };
  }, [url, template, fieldValues]);

  // Handle field value change
  const handleFieldChange = useCallback((key: string, value: string) => {
    setFieldValues((prev) => ({ ...prev, [key]: value }));
  }, []);

  // Handle form submission
  const handleSubmit = useCallback(() => {
    if (!template || !isValid) return;

    const prompt = buildPrompt(template.promptTemplate, url, fieldValues, template.fields);

    // Navigate to record page with query params
    const params = new URLSearchParams({
      url,
      ai_prompt: prompt,
    });

    handleClose();
    navigate(`/record?${params.toString()}`);
  }, [template, isValid, url, fieldValues, handleClose, navigate]);

  // Handle URL navigation (from URL bar)
  const handleUrlNavigate = useCallback((navigatedUrl: string) => {
    setUrl(navigatedUrl);
  }, []);

  if (!template) return null;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={handleClose}
      ariaLabel={`Configure ${template.name} template`}
      className="!overflow-visible"
    >
      <div className="relative">
        {/* Close button */}
        <button
          onClick={handleClose}
          className="absolute top-0 right-0 p-1 text-gray-400 hover:text-gray-200 transition-colors"
        >
          <X size={20} />
        </button>

        {/* Header */}
        <div className="flex items-center gap-3 mb-6">
          <div className="p-3 rounded-full bg-flow-accent/20 text-flow-accent">
            {template.icon}
          </div>
          <div>
            <h2 className="text-lg font-semibold text-surface">
              {template.name}
            </h2>
            <p className="text-sm text-gray-400">
              {template.description}
            </p>
          </div>
        </div>

        {/* URL Input Section - extra padding-bottom for dropdown space */}
        <div className="mb-4 pb-2">
          <label className="block text-sm font-medium text-gray-300 mb-2">
            <Globe size={14} className="inline mr-1.5" />
            Website URL <span className="text-red-400">*</span>
          </label>
          <div className="rounded-lg border border-gray-700 bg-gray-800 relative z-10">
            <BrowserUrlBar
              value={url}
              onChange={setUrl}
              onNavigate={handleUrlNavigate}
              onRefresh={() => {}}
              placeholder="Enter website URL (e.g., example.com)"
            />
          </div>
          <p className="text-xs text-gray-500 mt-1">
            The AI will navigate to this URL and perform the template actions
          </p>
        </div>

        {/* Dynamic Fields */}
        {template.fields.length > 0 && (
          <div className="space-y-4 mb-6">
            {template.fields.map((field) => (
              <div key={field.key}>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {field.label}
                  {field.required && <span className="text-red-400 ml-1">*</span>}
                </label>
                {field.type === 'textarea' ? (
                  <textarea
                    value={fieldValues[field.key] || ''}
                    onChange={(e) => handleFieldChange(field.key, e.target.value)}
                    placeholder={field.placeholder}
                    rows={3}
                    className="
                      w-full px-3 py-2 rounded-lg
                      bg-gray-800 border border-gray-700
                      text-surface placeholder-gray-500
                      focus:outline-none focus:ring-2 focus:ring-flow-accent focus:border-transparent
                      resize-none
                    "
                  />
                ) : (
                  <input
                    type="text"
                    value={fieldValues[field.key] || ''}
                    onChange={(e) => handleFieldChange(field.key, e.target.value)}
                    placeholder={field.placeholder}
                    className="
                      w-full px-3 py-2 rounded-lg
                      bg-gray-800 border border-gray-700
                      text-surface placeholder-gray-500
                      focus:outline-none focus:ring-2 focus:ring-flow-accent focus:border-transparent
                    "
                  />
                )}
                {!field.required && (
                  <p className="text-xs text-gray-500 mt-1">Optional</p>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Validation Message */}
        {!isValid && validationMessage && (
          <div className="mb-4 px-3 py-2 rounded-lg bg-amber-500/10 border border-amber-500/20">
            <p className="text-sm text-amber-400">{validationMessage}</p>
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-3">
          <button
            onClick={handleClose}
            className="
              flex-1 px-4 py-3 rounded-lg
              bg-gray-700 hover:bg-gray-600
              text-gray-300 font-medium
              transition-colors
            "
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!isValid}
            className="
              flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-lg
              bg-flow-accent hover:bg-flow-accent/80
              disabled:bg-gray-700 disabled:text-gray-500 disabled:cursor-not-allowed
              text-white font-medium
              transition-colors
            "
          >
            <Play size={18} />
            Start Automation
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default TemplateModal;
