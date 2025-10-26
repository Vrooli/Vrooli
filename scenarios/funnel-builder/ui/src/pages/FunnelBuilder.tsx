import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { DndContext, DragEndEvent, closestCenter } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { Plus, Save, Eye, Settings } from 'lucide-react'
import { useFunnelStore } from '../store/useFunnelStore'
import StepList from '../components/builder/StepList'
import StepEditor from '../components/builder/StepEditor'
import StepTypeSelector from '../components/builder/StepTypeSelector'
import toast from 'react-hot-toast'

const FunnelBuilder = () => {
  const { id } = useParams()
  const {
    currentFunnel,
    selectedStep,
    setCurrentFunnel,
    setSelectedStep,
    reorderSteps,
    addStep
  } = useFunnelStore()
  
  const [showStepSelector, setShowStepSelector] = useState(false)
  const [showSettings, setShowSettings] = useState(false)

  useEffect(() => {
    if (id) {
      // Load funnel by ID
      // For now, using mock data
      const mockFunnel = {
        id,
        name: 'Sample Funnel',
        slug: 'sample-funnel',
        description: 'A sample funnel for demonstration',
        steps: [
          {
            id: 'step-1',
            type: 'quiz' as const,
            position: 0,
            title: 'What are you looking for?',
            content: {
              question: 'What are you looking for?',
              options: [
                { id: 'opt-1', text: 'Lead Generation', icon: '📊' },
                { id: 'opt-2', text: 'Sales Automation', icon: '🚀' },
                { id: 'opt-3', text: 'Customer Support', icon: '💬' },
              ]
            }
          },
          {
            id: 'step-2',
            type: 'form' as const,
            position: 1,
            title: 'Get Started',
            content: {
              fields: [
                { id: 'field-1', type: 'text' as const, label: 'Full Name', required: true },
                { id: 'field-2', type: 'email' as const, label: 'Email', required: true },
                { id: 'field-3', type: 'tel' as const, label: 'Phone', required: false },
              ],
              submitText: 'Get Free Access'
            }
          }
        ],
        settings: {
          theme: { primaryColor: '#0ea5e9' },
          progressBar: true
        },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        status: 'draft' as const
      }
      setCurrentFunnel(mockFunnel)
    } else {
      // Create new funnel
      const newFunnel = {
        id: 'new-' + Date.now(),
        name: 'Untitled Funnel',
        slug: 'untitled-funnel',
        description: '',
        steps: [],
        settings: {
          theme: { primaryColor: '#0ea5e9' },
          progressBar: true
        },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        status: 'draft' as const
      }
      setCurrentFunnel(newFunnel)
    }
  }, [id])

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    
    if (over && active.id !== over.id) {
      const oldIndex = currentFunnel?.steps.findIndex((step) => step.id === active.id) ?? -1
      const newIndex = currentFunnel?.steps.findIndex((step) => step.id === over.id) ?? -1
      
      if (oldIndex !== -1 && newIndex !== -1) {
        reorderSteps(oldIndex, newIndex)
      }
    }
  }

  const handleSave = () => {
    // Save funnel to backend
    toast.success('Funnel saved successfully!')
  }

  const handlePreview = () => {
    window.open(`/preview/${currentFunnel?.id}`, '_blank')
  }

  if (!currentFunnel) {
    return <div className="p-8">Loading...</div>
  }

  return (
    <div className="flex flex-col md:h-screen md:flex-row">
      <div className="border-b border-gray-200 bg-white md:h-full md:w-80 md:border-b-0 md:border-r">
        <div className="flex items-center justify-between gap-3 px-4 py-4 md:block md:gap-0 md:p-4">
          <h3 className="text-base font-semibold text-gray-900 md:mb-4 md:text-lg">Funnel Steps</h3>
          <button
            onClick={() => setShowStepSelector(true)}
            className="btn btn-primary shrink-0 md:mt-4 md:w-full"
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Step
          </button>
        </div>

        <div className="max-h-[60vh] overflow-y-auto px-4 pb-4 md:flex-1 md:overflow-auto md:p-4">
          <DndContext
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={currentFunnel.steps.map((step) => step.id)}
              strategy={verticalListSortingStrategy}
            >
              <StepList
                steps={currentFunnel.steps}
                selectedStep={selectedStep}
                onSelectStep={setSelectedStep}
              />
            </SortableContext>
          </DndContext>
        </div>
      </div>

      <div className="flex flex-1 flex-col bg-gray-50">
        <div className="border-b border-gray-200 bg-white px-4 py-4 sm:px-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 className="text-lg font-bold text-gray-900 sm:text-xl">{currentFunnel.name}</h2>
              <p className="text-sm text-gray-600">{currentFunnel.description}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => setShowSettings(!showSettings)}
                className="btn btn-outline"
                aria-label="Funnel settings"
              >
                <Settings className="h-4 w-4" />
                <span className="ml-2 hidden text-sm sm:inline">Settings</span>
              </button>
              <button onClick={handlePreview} className="btn btn-secondary">
                <Eye className="mr-2 h-4 w-4" />
                Preview
              </button>
              <button onClick={handleSave} className="btn btn-primary">
                <Save className="mr-2 h-4 w-4" />
                Save
              </button>
            </div>
          </div>
        </div>

        <div className="flex-1 overflow-x-hidden md:overflow-auto">
          {selectedStep ? (
            <StepEditor step={selectedStep} />
          ) : (
            <div className="flex h-64 items-center justify-center px-4 text-gray-500 md:h-full">
              <div className="text-center">
                <p className="mb-2">Select a step to edit</p>
                <p className="text-sm">or add a new step to get started</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {showStepSelector && (
        <StepTypeSelector
          onSelect={(type) => {
            const newStep = {
              id: `step-${Date.now()}`,
              type,
              position: currentFunnel.steps.length,
              title: `New ${type} step`,
              content: {} as any
            }
            addStep(newStep)
            setShowStepSelector(false)
          }}
          onClose={() => setShowStepSelector(false)}
        />
      )}
    </div>
  )
}

export default FunnelBuilder
