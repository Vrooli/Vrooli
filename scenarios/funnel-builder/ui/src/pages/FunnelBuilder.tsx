import { useState, useEffect, useRef, useCallback, type KeyboardEvent, type FormEvent } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
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
import { FunnelStep, FunnelSettings } from '../types'
import { getFunnelTemplate } from '../data/funnelTemplates'
import {
  saveFunnelToCache,
  saveDraftFunnel,
  loadDraftFunnel,
  clearDraftFunnel,
  loadFunnelFromCache,
  removeFunnelFromCache
} from '../utils/funnelCache'
import {
  fetchFunnel,
  fetchProjects,
  createFunnel,
  updateFunnel,
  createProject
} from '../services/funnels'

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
  const navigate = useNavigate()
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const templateId = searchParams.get('template') ?? undefined
  const projectParam = searchParams.get('project') ?? searchParams.get('projectId') ?? undefined

  const {
    currentFunnel,
    selectedStep,
    setCurrentFunnel,
    setSelectedStep,
    reorderSteps,
    addStep,
    updateFunnelDetails,
    updateFunnelSettings,
    projects,
    setProjects,
    upsertFunnel
  } = useFunnelStore((state) => ({
    currentFunnel: state.currentFunnel,
    selectedStep: state.selectedStep,
    setCurrentFunnel: state.setCurrentFunnel,
    setSelectedStep: state.setSelectedStep,
    reorderSteps: state.reorderSteps,
    addStep: state.addStep,
    updateFunnelDetails: state.updateFunnelDetails,
    updateFunnelSettings: state.updateFunnelSettings,
    projects: state.projects,
    setProjects: state.setProjects,
    upsertFunnel: state.upsertFunnel,
  }))

  const [isInitializing, setIsInitializing] = useState(true)
  const [initializationError, setInitializationError] = useState<string | null>(null)
  const [reloadCounter, setReloadCounter] = useState(0)
  const [isSaving, setIsSaving] = useState(false)
  const [isNewFunnel, setIsNewFunnel] = useState(() => !id || id === 'new')
  const [showStepSelector, setShowStepSelector] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)
  const [isEditingName, setIsEditingName] = useState(false)
  const [nameDraft, setNameDraft] = useState('')
  const [isProjectModalOpen, setProjectModalOpen] = useState(false)
  const [projectDraft, setProjectDraft] = useState({ name: '', description: '' })
  const [projectModalError, setProjectModalError] = useState<string | null>(null)
  const [isSavingProject, setIsSavingProject] = useState(false)
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
    let cancelled = false

    const initialize = async () => {
      setInitializationError(null)
      setIsInitializing(true)

      const ensureProjects = async () => {
        if (projects.length > 0) {
          return
        }
        try {
          const data = await fetchProjects()
          if (!cancelled) {
            setProjects(data)
          }
        } catch (error) {
          throw new Error('Unable to load projects. Please try again.')
        }
      }

      const initExistingFunnel = async (funnelId: string) => {
        setIsNewFunnel(false)
        clearDraftFunnel()

        const cached = loadFunnelFromCache(funnelId)
        if (cached && !cancelled) {
          upsertFunnel(cached)
          setCurrentFunnel(cached)
          setSelectedStep(cached.steps[0] ?? null)
          return
        }

        const stored = useFunnelStore.getState().funnels.find((funnel) => funnel.id === funnelId)
        if (stored && !cancelled) {
          upsertFunnel(stored)
          setCurrentFunnel(stored)
          setSelectedStep(stored.steps[0] ?? null)
          return
        }

        try {
          const fetched = await fetchFunnel(funnelId)
          if (!cancelled) {
            upsertFunnel(fetched)
            setCurrentFunnel(fetched)
            setSelectedStep(fetched.steps[0] ?? null)
          }
        } catch (error) {
          throw new Error('Unable to load the requested funnel.')
        }
      }

      const initNewFunnel = () => {
        setIsNewFunnel(true)
        const draft = loadDraftFunnel()
        if (draft && !cancelled) {
          setCurrentFunnel(draft)
          setSelectedStep(draft.steps[0] ?? null)
          return
        }

        const now = new Date().toISOString()
        const uniqueSuffix = Date.now().toString()
        const preselectedProjectId = projectParam

        if (templateId) {
          const template = getFunnelTemplate(templateId)
          if (template) {
            const steps = template.funnel.steps.map((step, index) => ({
              ...JSON.parse(JSON.stringify(step)),
              id: `${step.id}-${uniqueSuffix}`,
              position: index,
            }))
            const templateSettings = JSON.parse(JSON.stringify(template.funnel.settings))

            const templateFunnel = {
              id: `new-${uniqueSuffix}`,
              name: template.funnel.name,
              slug: `${template.id}-${uniqueSuffix}`,
              description: template.funnel.description,
              steps,
              settings: templateSettings,
              projectId: preselectedProjectId ?? null,
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
          projectId: preselectedProjectId ?? null,
          createdAt: now,
          updatedAt: now,
          status: 'draft' as const
        }

        setCurrentFunnel(newFunnel)
        setSelectedStep(null)
      }

      try {
        await ensureProjects()

        if (id && id !== 'new') {
          await initExistingFunnel(id)
        } else {
          initNewFunnel()
        }
      } catch (error) {
        if (!cancelled) {
          console.error('Failed to initialize funnel builder', error)
          setInitializationError(error instanceof Error ? error.message : 'Unable to load funnel builder.')
        }
      } finally {
        if (!cancelled) {
          setIsInitializing(false)
        }
      }
    }

    void initialize()

    return () => {
      cancelled = true
    }
  }, [id, templateId, projectParam, projects.length, setCurrentFunnel, setProjects, setSelectedStep, upsertFunnel, reloadCounter])

  const handleRetryInitialization = useCallback(() => {
    setReloadCounter((count) => count + 1)
  }, [])

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

  const handleSave = useCallback(async () => {
    if (!currentFunnel) {
      return
    }

    if (!currentFunnel.projectId) {
      toast.error('Select a project before saving your funnel.')
      return
    }

    setIsSaving(true)

    try {
      if (isNewFunnel || !id || id === 'new' || currentFunnel.id.startsWith('new-')) {
        const created = await createFunnel({
          name: currentFunnel.name,
          description: currentFunnel.description,
          slug: currentFunnel.slug,
          steps: currentFunnel.steps,
          settings: currentFunnel.settings,
          status: currentFunnel.status,
          tenantId: currentFunnel.tenantId ?? undefined,
          projectId: currentFunnel.projectId,
        })

        upsertFunnel(created)
        setCurrentFunnel(created)
        setSelectedStep(created.steps[0] ?? null)
        clearDraftFunnel()
        removeFunnelFromCache(currentFunnel.id)
        setIsNewFunnel(false)
        toast.success('Funnel created and linked to your project.')
        navigate(`/builder/${created.id}`, { replace: true })
      } else {
        const updated = await updateFunnel(currentFunnel.id, {
          name: currentFunnel.name,
          description: currentFunnel.description,
          steps: currentFunnel.steps,
          settings: currentFunnel.settings,
          status: currentFunnel.status,
          projectId: currentFunnel.projectId,
        })

        upsertFunnel(updated)
        setCurrentFunnel(updated)
        const nextSelectedStep = selectedStep
          ? updated.steps.find((step) => step.id === selectedStep.id) ?? updated.steps[0] ?? null
          : updated.steps[0] ?? null
        setSelectedStep(nextSelectedStep)
        toast.success('Funnel saved successfully!')
      }
      try {
        const refreshedProjects = await fetchProjects()
        setProjects(refreshedProjects)
      } catch (refreshError) {
        console.error('Failed to refresh projects after saving funnel', refreshError)
      }
    } catch (error) {
      console.error('Failed to save funnel', error)
      toast.error(error instanceof Error ? error.message : 'Unable to save funnel.')
    } finally {
      setIsSaving(false)
    }
  }, [currentFunnel, id, isNewFunnel, navigate, selectedStep, setCurrentFunnel, setProjects, setSelectedStep, upsertFunnel])

  useEffect(() => {
    if (currentFunnel) {
      setNameDraft(currentFunnel.name)
      saveFunnelToCache(currentFunnel)
      if (isNewFunnel || currentFunnel.id.startsWith('new-')) {
        saveDraftFunnel(currentFunnel)
      }
    }
  }, [currentFunnel, isNewFunnel])

  const handleSettingsSave = useCallback(
    async (settings: FunnelSettings, scope: 'funnel' | 'project') => {
      if (!currentFunnel) {
        return
      }

      const cloneSettings = () => JSON.parse(JSON.stringify(settings)) as FunnelSettings
      const normalizedSettings = cloneSettings()
      updateFunnelSettings(normalizedSettings)

      if (scope === 'project') {
        const projectId = currentFunnel.projectId
        if (!projectId) {
          toast.success('Funnel settings updated.')
          setShowSettings(false)
          return
        }

        const sourceProject = projects.find((project) => project.id === projectId)
        if (!sourceProject) {
          toast.error('Unable to locate the selected project. Settings were only applied to this funnel.')
          setShowSettings(false)
          return
        }

        const timestamp = new Date().toISOString()
        const updatedProjects = projects.map((project) => {
          if (project.id !== projectId) {
            return project
          }

          const updatedFunnels = project.funnels.map((funnel) => ({
            ...funnel,
            settings: cloneSettings(),
            updatedAt: timestamp
          }))

          return { ...project, funnels: updatedFunnels }
        })

        setProjects(updatedProjects)

        try {
          const updatedResults = await Promise.all(
            sourceProject.funnels.map((funnel) =>
              updateFunnel(funnel.id, {
                settings: cloneSettings(),
                projectId: funnel.projectId ?? projectId
              })
            )
          )

          updatedResults.forEach((funnel) => upsertFunnel(funnel))

          try {
            const refreshedProjects = await fetchProjects()
            setProjects(refreshedProjects)
          } catch (refreshError) {
            console.error('Failed to refresh projects after applying settings', refreshError)
          }

          toast.success('Applied settings to every funnel in this project.')
          setShowSettings(false)
        } catch (error) {
          console.error('Failed to apply settings to project funnels', error)
          toast.error('Failed to update every funnel. Only this funnel reflects the new settings.')
          try {
            const refreshedProjects = await fetchProjects()
            setProjects(refreshedProjects)
          } catch (refreshError) {
            console.error('Failed to refresh projects after settings error', refreshError)
          }
        }

        return
      }

      toast.success('Funnel settings updated.')
      setShowSettings(false)
    },
    [currentFunnel, projects, setProjects, updateFunnelSettings, upsertFunnel]
  )

  const openProjectModal = () => {
    setProjectModalError(null)
    setProjectDraft({ name: '', description: '' })
    setProjectModalOpen(true)
  }

  const closeProjectModal = () => {
    setProjectModalOpen(false)
    setProjectModalError(null)
    setIsSavingProject(false)
    setProjectDraft({ name: '', description: '' })
  }

  const handleProjectSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedName = projectDraft.name.trim()
    const trimmedDescription = projectDraft.description.trim()

    if (!trimmedName) {
      setProjectModalError('Project name is required')
      return
    }

    setIsSavingProject(true)
    setProjectModalError(null)

    try {
      const newProject = await createProject({
        name: trimmedName,
        description: trimmedDescription !== '' ? trimmedDescription : undefined
      })

      const refreshedProjects = await fetchProjects()
      setProjects(refreshedProjects)
      updateFunnelDetails({ projectId: newProject.id })
      toast.success('Project created.')
      closeProjectModal()
    } catch (error) {
      console.error('Failed to create project', error)
      setProjectModalError('Unable to create project. Try again in a moment.')
      toast.error('Failed to create project')
    } finally {
      setIsSavingProject(false)
    }
  }

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

  if (isInitializing) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center bg-gray-50 p-8">
        <div className="rounded-lg border border-gray-200 bg-white px-6 py-8 text-center shadow-sm">
          <p className="text-sm font-medium text-gray-700">Preparing funnel builder…</p>
        </div>
      </div>
    )
  }

  if (initializationError) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center bg-gray-50 p-8">
        <div className="max-w-md space-y-4 rounded-lg border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">
          <div>
            <h2 className="text-lg font-semibold">We couldn't load the funnel</h2>
            <p className="mt-1 text-sm leading-relaxed">{initializationError}</p>
          </div>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={handleRetryInitialization}
              className="btn btn-primary"
            >
              Try Again
            </button>
          </div>
        </div>
      </div>
    )
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
                <div className="mt-3 space-y-2">
                  <label className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                    Project
                  </label>
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                    <select
                      value={currentFunnel.projectId ?? ''}
                      onChange={(event) =>
                        updateFunnelDetails({
                          projectId: event.target.value ? event.target.value : null,
                        })
                      }
                      className="input sm:max-w-xs"
                    >
                      <option value="">Select a project…</option>
                      {projects.map((project) => (
                        <option key={project.id} value={project.id}>
                          {project.name}
                        </option>
                      ))}
                    </select>
                    <button
                      type="button"
                      onClick={openProjectModal}
                      className="btn btn-outline sm:w-auto"
                    >
                      <Plus className="mr-2 h-4 w-4" />
                      New Project
                    </button>
                  </div>
                  {!currentFunnel.projectId && (
                    <p className="text-xs text-amber-600">
                      Select a project to keep this funnel organized and enable saving.
                    </p>
                  )}
                </div>
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
                onClick={() => void handleSave()}
                className="icon-btn icon-btn-primary"
                aria-label={isSaving ? 'Saving funnel' : 'Save funnel'}
                title={isSaving ? 'Saving funnel…' : 'Save funnel'}
                disabled={isSaving}
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
          canApplyToProject={Boolean(currentFunnel.projectId)}
          onClose={() => setShowSettings(false)}
          onSave={handleSettingsSave}
        />
      )}
      {isProjectModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4"
          role="dialog"
          aria-modal="true"
          onClick={closeProjectModal}
        >
          <div
            className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-semibold text-gray-900">Create Project</h3>
                <p className="text-sm text-gray-500">Organize funnels by grouping them into a dedicated project.</p>
              </div>
              <button
                type="button"
                onClick={closeProjectModal}
                className="icon-btn icon-btn-ghost h-9 w-9"
                aria-label="Close create project dialog"
              >
                <X className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
            <form onSubmit={handleProjectSubmit} className="mt-4 space-y-4">
              <div>
                <label htmlFor="builder-project-name" className="block text-sm font-medium text-gray-700">
                  Project name
                </label>
                <input
                  id="builder-project-name"
                  type="text"
                  value={projectDraft.name}
                  onChange={(event) => setProjectDraft((prev) => ({ ...prev, name: event.target.value }))}
                  className="input mt-1"
                  placeholder="e.g. Spring Launch"
                  required
                />
              </div>
              <div>
                <label htmlFor="builder-project-description" className="block text-sm font-medium text-gray-700">
                  Description (optional)
                </label>
                <textarea
                  id="builder-project-description"
                  value={projectDraft.description}
                  onChange={(event) => setProjectDraft((prev) => ({ ...prev, description: event.target.value }))}
                  className="input mt-1 min-h-[80px]"
                  placeholder="Give teammates context about this collection of funnels."
                />
              </div>
              {projectModalError && (
                <p className="text-sm text-amber-600">{projectModalError}</p>
              )}
              <div className="flex justify-end gap-2">
                <button type="button" onClick={closeProjectModal} className="btn btn-secondary">
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={isSavingProject}>
                  {isSavingProject ? 'Saving…' : 'Create Project'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default FunnelBuilder
