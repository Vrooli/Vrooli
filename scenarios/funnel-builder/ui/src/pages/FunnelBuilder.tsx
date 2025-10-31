import { useState, useEffect, useRef, type KeyboardEvent } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { DndContext, DragEndEvent, closestCenter } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import {
  Eye,
  Layers,
  Menu,
  PanelLeftOpen,
  Plus,
  Save,
  Settings,
  X,
} from 'lucide-react'
import { useFunnelStore } from '../store/useFunnelStore'
import StepList from '../components/builder/StepList'
import StepEditor from '../components/builder/StepEditor'
import StepTypeSelector from '../components/builder/StepTypeSelector'
import FunnelSettingsModal from '../components/builder/FunnelSettingsModal'
import toast from 'react-hot-toast'
import { FunnelStep } from '../types'
import { getFunnelTemplate } from '../data/funnelTemplates'
import { saveFunnelToCache } from '../utils/funnelCache'

const buildPreviewUrl = (funnelId?: string) => {
  if (!funnelId) {
    return ''
  }

  const previewPath = `/preview/${funnelId}`

  if (typeof window === 'undefined') {
    return previewPath
  }

  const basePath = window.__FUNNEL_BUILDER_BASE_PATH__
  if (!basePath || basePath === '/' || basePath === '') {
    return previewPath
  }

  const normalizedBase = basePath.endsWith('/') ? basePath.slice(0, -1) : basePath
  return `${normalizedBase}${previewPath}`
}

const FunnelBuilder = () => {
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const templateId = searchParams.get('template') ?? undefined

  const {
    currentFunnel,
    selectedStep,
    setCurrentFunnel,
    setSelectedStep,
    reorderSteps,
    addStep,
    updateFunnelDetails,
    updateFunnelSettings
  } = useFunnelStore()
  
  const [showStepSelector, setShowStepSelector] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)
  const [isEditingName, setIsEditingName] = useState(false)
  const [nameDraft, setNameDraft] = useState('')
  const nameInputRef = useRef<HTMLInputElement | null>(null)
  const cancelNameEditRef = useRef(false)

  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth >= 768) {
        setIsMobileSidebarOpen(false)
      }
    }

    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  useEffect(() => {
    if (id && id !== 'new') {
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
      setSelectedStep(mockFunnel.steps[0])
      return
    }

    const now = new Date().toISOString()
    const uniqueSuffix = Date.now().toString()

    if (templateId) {
      const template = getFunnelTemplate(templateId)

      if (template) {
        const steps = template.funnel.steps.map((step, index) => ({
          ...step,
          id: `${step.id}-${uniqueSuffix}`,
          position: index,
          content: JSON.parse(JSON.stringify(step.content)) as typeof step.content
        }))

        const templateFunnel = {
          id: `new-${uniqueSuffix}`,
          name: template.funnel.name,
          slug: `${template.id}-${uniqueSuffix}`,
          description: template.funnel.description,
          steps,
          settings: {
            ...template.funnel.settings
          },
          createdAt: now,
          updatedAt: now,
          status: 'draft' as const
        }

        setCurrentFunnel(templateFunnel)
        setSelectedStep(steps[0] ?? null)
        return
      }
    }

    const newFunnel = {
      id: `new-${uniqueSuffix}`,
      name: 'Untitled Funnel',
      slug: `untitled-${uniqueSuffix}`,
      description: '',
      steps: [],
      settings: {
        theme: { primaryColor: '#0ea5e9' },
        progressBar: true
      },
      createdAt: now,
      updatedAt: now,
      status: 'draft' as const
    }
    setCurrentFunnel(newFunnel)
    setSelectedStep(null)
  }, [id, templateId, setCurrentFunnel, setSelectedStep])

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

  const handleSelectStep = (step: FunnelStep) => {
    setSelectedStep(step)
    setIsMobileSidebarOpen(false)
  }

  const openStepSelector = (source: 'mobile' | 'desktop') => {
    setShowStepSelector(true)
    if (source === 'mobile') {
      setIsMobileSidebarOpen(false)
    }
  }

  const handleSave = () => {
    // Save funnel to backend
    toast.success('Funnel saved successfully!')
  }

  useEffect(() => {
    if (currentFunnel) {
      setNameDraft(currentFunnel.name)
      saveFunnelToCache(currentFunnel)
    }
  }, [currentFunnel])

  useEffect(() => {
    if (isEditingName) {
      const focusField = () => {
        nameInputRef.current?.focus()
        nameInputRef.current?.select()
      }

      if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
        window.requestAnimationFrame(focusField)
      } else {
        focusField()
      }
    }
  }, [isEditingName])

  const beginEditingName = () => {
    cancelNameEditRef.current = false
    setNameDraft(currentFunnel?.name ?? '')
    setIsEditingName(true)
  }

  const cancelNameEdit = () => {
    cancelNameEditRef.current = true
    setNameDraft(currentFunnel?.name ?? '')
    setIsEditingName(false)
  }

  const commitName = () => {
    const trimmed = nameDraft.trim()
    const nextName = trimmed.length > 0 ? trimmed : 'Untitled Funnel'
    updateFunnelDetails({ name: nextName })
    setNameDraft(nextName)
    cancelNameEditRef.current = false
    setIsEditingName(false)
  }

  const handleNameBlur = () => {
    if (cancelNameEditRef.current) {
      cancelNameEditRef.current = false
      return
    }
    commitName()
  }

  const handleNameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault()
      commitName()
      return
    }

    if (event.key === 'Escape') {
      event.preventDefault()
      cancelNameEdit()
    }
  }

  const handlePreview = () => {
    const previewUrl = buildPreviewUrl(currentFunnel?.id)
    if (!previewUrl) {
      toast.error('Unable to open preview – funnel is not ready.')
      return
    }

    if (currentFunnel) {
      saveFunnelToCache(currentFunnel)
    }

    window.open(previewUrl, '_blank', 'noopener,noreferrer')
  }

  if (!currentFunnel) {
    return <div className="p-8">Loading...</div>
  }

  const sidebarList = (
    <div className="flex-1 overflow-y-auto px-4 pb-4 md:p-4">
      <DndContext collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext
          items={currentFunnel.steps.map((step) => step.id)}
          strategy={verticalListSortingStrategy}
        >
          <StepList
            steps={currentFunnel.steps}
            selectedStep={selectedStep}
            onSelectStep={handleSelectStep}
          />
        </SortableContext>
      </DndContext>
    </div>
  )

  return (
    <div className="flex flex-col md:h-screen md:flex-row">
      {isMobileSidebarOpen && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/40"
            onClick={() => setIsMobileSidebarOpen(false)}
          />
          <aside className="fixed inset-y-0 left-0 z-50 flex w-72 max-w-full flex-col bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-gray-200 px-4 py-4">
              <h3 className="text-lg font-semibold text-gray-900">Funnel Steps</h3>
              <button
                type="button"
                onClick={() => setIsMobileSidebarOpen(false)}
                className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-gray-200 text-gray-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
                aria-label="Close step navigation"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="px-4 py-4">
              <button
                onClick={() => openStepSelector('mobile')}
                className="btn btn-primary w-full"
              >
                <Plus className="mr-2 h-4 w-4" />
                Add Step
              </button>
            </div>
            {sidebarList}
          </aside>
        </>
      )}

      <aside
        className={`hidden md:flex md:flex-col md:border-r md:border-gray-200 md:bg-white transition-all duration-200 ${
          isSidebarCollapsed ? 'md:w-16' : 'md:w-80'
        }`}
      >
        {isSidebarCollapsed ? (
          <div className="flex h-full items-center justify-center">
            <button
              type="button"
              onClick={() => setIsSidebarCollapsed(false)}
              className="flex items-center justify-center"
              aria-label="Expand step navigation"
              title="Expand step navigation"
            >
              <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-100 text-primary-600 shadow-sm">
                <Layers className="h-5 w-5" aria-hidden="true" />
              </span>
            </button>
          </div>
        ) : (
          <>
            <div className="border-b border-gray-200 px-4 py-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary-100 text-primary-600">
                    <Layers className="h-5 w-5" aria-hidden="true" />
                    <span className="sr-only">Funnel Builder icon</span>
                  </span>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">Funnel Steps</h3>
                    <p className="text-xs text-gray-500">Drag to reorder steps</p>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => setIsSidebarCollapsed(true)}
                  className="icon-btn icon-btn-ghost h-9 w-9"
                  aria-label="Collapse step navigation"
                  title="Collapse step navigation"
                >
                  <PanelLeftOpen className="h-4 w-4" aria-hidden="true" />
                </button>
              </div>
            </div>
            <div className="px-4 py-4">
              <button
                onClick={() => openStepSelector('desktop')}
                className="btn btn-primary w-full"
              >
                <Plus className="mr-2 h-4 w-4" />
                Add Step
              </button>
            </div>
            {sidebarList}
          </>
        )}
      </aside>

      <div className="flex flex-1 flex-col bg-gray-50">
        <div className="border-b border-gray-200 bg-white px-4 py-4 sm:px-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setIsMobileSidebarOpen(true)}
                className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-gray-200 text-gray-600 shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 md:hidden"
                aria-label="Open step navigation"
              >
                <Menu className="h-5 w-5" />
              </button>
              <div>
                {isEditingName ? (
                  <input
                    ref={nameInputRef}
                    value={nameDraft}
                    onChange={(event) => setNameDraft(event.target.value)}
                    onBlur={handleNameBlur}
                    onKeyDown={handleNameKeyDown}
                    className="w-full max-w-md border-b border-transparent bg-transparent text-lg font-bold text-gray-900 focus:border-primary-500 focus:outline-none focus:ring-0 sm:text-xl"
                    aria-label="Funnel name"
                  />
                ) : (
                  <button
                    type="button"
                    onClick={beginEditingName}
                    className="text-left text-lg font-bold text-gray-900 transition-colors hover:text-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 sm:text-xl"
                    aria-label="Edit funnel name"
                    title="Click to rename funnel"
                  >
                    {currentFunnel.name}
                  </button>
                )}
                <p className="text-sm text-gray-600">{currentFunnel.description}</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowSettings(true)}
                className="icon-btn icon-btn-ghost"
                aria-label="Funnel settings"
                title="Funnel settings"
              >
                <Settings className="h-4 w-4" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={handlePreview}
                className="icon-btn icon-btn-secondary"
                aria-label="Preview funnel"
                title="Preview funnel"
              >
                <Eye className="h-4 w-4" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={handleSave}
                className="icon-btn icon-btn-primary"
                aria-label="Save funnel"
                title="Save funnel"
              >
                <Save className="h-4 w-4" aria-hidden="true" />
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
      {showSettings && currentFunnel && (
        <FunnelSettingsModal
          settings={currentFunnel.settings}
          onClose={() => setShowSettings(false)}
          onSave={(settings) => {
            updateFunnelSettings(settings)
            toast.success('Funnel settings updated')
            setShowSettings(false)
          }}
        />
      )}
    </div>
  )
}

export default FunnelBuilder
