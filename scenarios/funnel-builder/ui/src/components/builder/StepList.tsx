import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { FunnelStep } from '../../types'
import { GripVertical, FileText, HelpCircle, MousePointer, Zap, Trash2 } from 'lucide-react'
import { useFunnelStore } from '../../store/useFunnelStore'

interface StepListProps {
  steps: FunnelStep[]
  selectedStep: FunnelStep | null
  onSelectStep: (step: FunnelStep) => void
}

interface StepItemProps {
  step: FunnelStep
  isSelected: boolean
  onSelect: () => void
  onDelete: () => void
}

const stepIcons = {
  quiz: HelpCircle,
  form: FileText,
  content: FileText,
  cta: MousePointer,
}

const StepItem = ({ step, isSelected, onSelect, onDelete }: StepItemProps) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: step.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  const Icon = stepIcons[step.type]

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`group relative p-3 mb-2 bg-white rounded-lg border-2 transition-all cursor-pointer ${
        isSelected
          ? 'border-primary-500 shadow-md'
          : 'border-gray-200 hover:border-gray-300'
      } ${isDragging ? 'opacity-50' : ''}`}
      onClick={onSelect}
    >
      <div className="flex items-center gap-3">
        <div
          {...attributes}
          {...listeners}
          className="cursor-grab active:cursor-grabbing"
        >
          <GripVertical className="w-4 h-4 text-gray-400" />
        </div>
        
        <Icon className="w-4 h-4 text-gray-600" />
        
        <div className="flex-1">
          <p className="font-medium text-sm text-gray-900">{step.title}</p>
          <p className="text-xs text-gray-500 capitalize">{step.type}</p>
        </div>
        
        <button
          onClick={(e) => {
            e.stopPropagation()
            onDelete()
          }}
          className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-50 rounded"
        >
          <Trash2 className="w-4 h-4 text-red-500" />
        </button>
      </div>
    </div>
  )
}

const StepList = ({ steps, selectedStep, onSelectStep }: StepListProps) => {
  const { deleteStep } = useFunnelStore()

  if (steps.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        <Zap className="w-12 h-12 mx-auto mb-3 text-gray-300" />
        <p className="text-sm">No steps yet</p>
        <p className="text-xs mt-1">Click "Add Step" to get started</p>
      </div>
    )
  }

  return (
    <div>
      {steps.map((step) => (
        <StepItem
          key={step.id}
          step={step}
          isSelected={selectedStep?.id === step.id}
          onSelect={() => onSelectStep(step)}
          onDelete={() => deleteStep(step.id)}
        />
      ))}
    </div>
  )
}

export default StepList