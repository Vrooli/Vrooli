import { FunnelStep, QuizContent, FormContent, ContentStep, CTAContent } from '../../types'
import { useFunnelStore } from '../../store/useFunnelStore'
import QuizEditor from './editors/QuizEditor'
import FormEditor from './editors/FormEditor'
import ContentEditor from './editors/ContentEditor'
import CTAEditor from './editors/CTAEditor'

interface StepEditorProps {
  step: FunnelStep
}

const StepEditor = ({ step }: StepEditorProps) => {
  const { updateStep } = useFunnelStore()

  const handleUpdate = (updates: Partial<FunnelStep>) => {
    updateStep(step.id, updates)
  }

  return (
    <div className="p-6">
      <div className="max-w-3xl mx-auto">
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Step Title
          </label>
          <input
            type="text"
            value={step.title}
            onChange={(e) => handleUpdate({ title: e.target.value })}
            className="input w-full"
            placeholder="Enter step title"
          />
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          {step.type === 'quiz' && (
            <QuizEditor
              content={step.content as QuizContent}
              onChange={(content) => handleUpdate({ content })}
            />
          )}
          
          {step.type === 'form' && (
            <FormEditor
              content={step.content as FormContent}
              onChange={(content) => handleUpdate({ content })}
            />
          )}
          
          {step.type === 'content' && (
            <ContentEditor
              content={step.content as ContentStep}
              onChange={(content) => handleUpdate({ content })}
            />
          )}
          
          {step.type === 'cta' && (
            <CTAEditor
              content={step.content as CTAContent}
              onChange={(content) => handleUpdate({ content })}
            />
          )}
        </div>
      </div>
    </div>
  )
}

export default StepEditor