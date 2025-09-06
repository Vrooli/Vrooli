import { FormContent, FormField } from '../../../types'
import { Plus, Trash2, ChevronDown } from 'lucide-react'

interface FormEditorProps {
  content: FormContent
  onChange: (content: FormContent) => void
}

const fieldTypes = [
  { value: 'text', label: 'Text' },
  { value: 'email', label: 'Email' },
  { value: 'tel', label: 'Phone' },
  { value: 'number', label: 'Number' },
  { value: 'textarea', label: 'Text Area' },
  { value: 'select', label: 'Dropdown' },
  { value: 'checkbox', label: 'Checkbox' },
]

const FormEditor = ({ content, onChange }: FormEditorProps) => {
  const handleFieldChange = (index: number, updates: Partial<FormField>) => {
    const newFields = [...(content.fields || [])]
    newFields[index] = { ...newFields[index], ...updates }
    onChange({ ...content, fields: newFields })
  }

  const addField = () => {
    const newField: FormField = {
      id: `field-${Date.now()}`,
      type: 'text',
      label: 'New Field',
      required: false
    }
    onChange({ ...content, fields: [...(content.fields || []), newField] })
  }

  const removeField = (index: number) => {
    const newFields = content.fields?.filter((_, i) => i !== index) || []
    onChange({ ...content, fields: newFields })
  }

  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Form Fields</h3>

      <div className="space-y-4 mb-6">
        {content.fields?.map((field, index) => (
          <div key={field.id} className="p-4 bg-gray-50 rounded-lg">
            <div className="flex justify-between items-start mb-3">
              <div className="flex-1 grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">
                    Label
                  </label>
                  <input
                    type="text"
                    value={field.label}
                    onChange={(e) => handleFieldChange(index, { label: e.target.value })}
                    className="input w-full text-sm"
                    placeholder="Field label"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">
                    Type
                  </label>
                  <select
                    value={field.type}
                    onChange={(e) => handleFieldChange(index, { type: e.target.value as FormField['type'] })}
                    className="input w-full text-sm"
                  >
                    {fieldTypes.map((type) => (
                      <option key={type.value} value={type.value}>
                        {type.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <button
                onClick={() => removeField(index)}
                className="ml-3 p-1.5 hover:bg-red-50 rounded transition-colors"
              >
                <Trash2 className="w-4 h-4 text-red-500" />
              </button>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  Placeholder
                </label>
                <input
                  type="text"
                  value={field.placeholder || ''}
                  onChange={(e) => handleFieldChange(index, { placeholder: e.target.value })}
                  className="input w-full text-sm"
                  placeholder="Placeholder text"
                />
              </div>
              <div className="flex items-end">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={field.required || false}
                    onChange={(e) => handleFieldChange(index, { required: e.target.checked })}
                    className="rounded border-gray-300"
                  />
                  <span className="text-sm text-gray-700">Required</span>
                </label>
              </div>
            </div>

            {field.type === 'select' && (
              <div className="mt-3">
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  Options (comma-separated)
                </label>
                <input
                  type="text"
                  value={field.options?.join(', ') || ''}
                  onChange={(e) => handleFieldChange(index, { 
                    options: e.target.value.split(',').map(o => o.trim()).filter(Boolean)
                  })}
                  className="input w-full text-sm"
                  placeholder="Option 1, Option 2, Option 3"
                />
              </div>
            )}
          </div>
        ))}
      </div>

      <button
        onClick={addField}
        className="btn btn-secondary w-full"
      >
        <Plus className="w-4 h-4 mr-2" />
        Add Field
      </button>

      <div className="mt-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Submit Button Text
        </label>
        <input
          type="text"
          value={content.submitText || 'Submit'}
          onChange={(e) => onChange({ ...content, submitText: e.target.value })}
          className="input w-full"
          placeholder="Submit button text"
        />
      </div>
    </div>
  )
}

export default FormEditor