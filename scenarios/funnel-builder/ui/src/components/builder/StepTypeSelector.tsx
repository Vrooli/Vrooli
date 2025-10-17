import { StepType } from '../../types'
import { HelpCircle, FileText, Image, MousePointer, X } from 'lucide-react'

interface StepTypeSelectorProps {
  onSelect: (type: StepType) => void
  onClose: () => void
}

const stepTypes = [
  {
    type: 'quiz' as StepType,
    icon: HelpCircle,
    title: 'Quiz',
    description: 'Multiple choice questions to segment leads',
    color: 'bg-blue-50 text-blue-600 hover:bg-blue-100'
  },
  {
    type: 'form' as StepType,
    icon: FileText,
    title: 'Form',
    description: 'Collect lead information with custom fields',
    color: 'bg-green-50 text-green-600 hover:bg-green-100'
  },
  {
    type: 'content' as StepType,
    icon: Image,
    title: 'Content',
    description: 'Display text, images, or videos',
    color: 'bg-purple-50 text-purple-600 hover:bg-purple-100'
  },
  {
    type: 'cta' as StepType,
    icon: MousePointer,
    title: 'Call to Action',
    description: 'Final conversion step with strong CTA',
    color: 'bg-orange-50 text-orange-600 hover:bg-orange-100'
  }
]

const StepTypeSelector = ({ onSelect, onClose }: StepTypeSelectorProps) => {
  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4">
        <div className="flex justify-between items-center p-6 border-b border-gray-200">
          <h3 className="text-xl font-semibold text-gray-900">Choose Step Type</h3>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>
        
        <div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-4">
          {stepTypes.map((stepType) => (
            <button
              key={stepType.type}
              onClick={() => onSelect(stepType.type)}
              className={`p-6 rounded-lg border-2 border-gray-200 hover:border-primary-400 transition-all text-left group ${stepType.color}`}
            >
              <div className="flex items-start gap-4">
                <div className="p-3 rounded-lg bg-white shadow-sm">
                  <stepType.icon className="w-6 h-6" />
                </div>
                <div className="flex-1">
                  <h4 className="font-semibold text-gray-900 mb-1">{stepType.title}</h4>
                  <p className="text-sm text-gray-600">{stepType.description}</p>
                </div>
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

export default StepTypeSelector