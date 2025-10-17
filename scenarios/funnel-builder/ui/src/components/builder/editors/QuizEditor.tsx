import { QuizContent, QuizOption } from '../../../types'
import { Plus, Trash2, Image } from 'lucide-react'

interface QuizEditorProps {
  content: QuizContent
  onChange: (content: QuizContent) => void
}

const QuizEditor = ({ content, onChange }: QuizEditorProps) => {
  const handleQuestionChange = (question: string) => {
    onChange({ ...content, question })
  }

  const handleOptionChange = (index: number, updates: Partial<QuizOption>) => {
    const newOptions = [...(content.options || [])]
    newOptions[index] = { ...newOptions[index], ...updates }
    onChange({ ...content, options: newOptions })
  }

  const addOption = () => {
    const newOption: QuizOption = {
      id: `opt-${Date.now()}`,
      text: 'New Option'
    }
    onChange({ ...content, options: [...(content.options || []), newOption] })
  }

  const removeOption = (index: number) => {
    const newOptions = content.options?.filter((_, i) => i !== index) || []
    onChange({ ...content, options: newOptions })
  }

  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Quiz Question</h3>
      
      <div className="mb-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Question
        </label>
        <input
          type="text"
          value={content.question || ''}
          onChange={(e) => handleQuestionChange(e.target.value)}
          className="input w-full"
          placeholder="What would you like to ask?"
        />
      </div>

      <div className="mb-4">
        <div className="flex justify-between items-center mb-3">
          <label className="text-sm font-medium text-gray-700">Answer Options</label>
          <button
            onClick={addOption}
            className="btn btn-secondary text-sm py-1 px-3"
          >
            <Plus className="w-4 h-4 mr-1" />
            Add Option
          </button>
        </div>

        <div className="space-y-3">
          {content.options?.map((option, index) => (
            <div key={option.id} className="flex gap-3 items-center">
              <div className="flex-1 flex gap-2">
                <input
                  type="text"
                  value={option.icon || ''}
                  onChange={(e) => handleOptionChange(index, { icon: e.target.value })}
                  className="input w-16 text-center"
                  placeholder="🎯"
                  title="Emoji or icon"
                />
                <input
                  type="text"
                  value={option.text}
                  onChange={(e) => handleOptionChange(index, { text: e.target.value })}
                  className="input flex-1"
                  placeholder="Option text"
                />
              </div>
              {option.image && (
                <div className="w-10 h-10 bg-gray-100 rounded flex items-center justify-center">
                  <Image className="w-5 h-5 text-gray-400" />
                </div>
              )}
              <button
                onClick={() => removeOption(index)}
                className="p-2 hover:bg-red-50 rounded transition-colors"
              >
                <Trash2 className="w-4 h-4 text-red-500" />
              </button>
            </div>
          ))}
        </div>
      </div>

      <div className="mt-4 p-4 bg-gray-50 rounded-lg">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={content.multiSelect || false}
            onChange={(e) => onChange({ ...content, multiSelect: e.target.checked })}
            className="rounded border-gray-300"
          />
          <span className="text-sm text-gray-700">Allow multiple selections</span>
        </label>
      </div>
    </div>
  )
}

export default QuizEditor