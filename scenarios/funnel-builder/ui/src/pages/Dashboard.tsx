import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, TrendingUp, Users, Target, Clock, AlertTriangle, RefreshCw, Pencil, Folder } from 'lucide-react'
import toast from 'react-hot-toast'
import { useFunnelStore } from '../store/useFunnelStore'
import type { Project } from '../types'
import { fetchProjects, createProject, updateProject } from '../services/funnels'

const isAbortError = (error: unknown) => error instanceof DOMException && error.name === 'AbortError'

const formatDateTime = (input?: string | null, options?: Intl.DateTimeFormatOptions) => {
  if (!input) {
    return '—'
  }

  const date = new Date(input)
  if (Number.isNaN(date.getTime())) {
    return '—'
  }

  return date.toLocaleString(undefined, options)
}

const Dashboard = () => {
  const navigate = useNavigate()
  const { projects, funnels, isLoading, error, setProjects, setLoading, setError } = useFunnelStore((state) => ({
    projects: state.projects,
    funnels: state.funnels,
    isLoading: state.isLoading,
    error: state.error,
    setProjects: state.setProjects,
    setLoading: state.setLoading,
    setError: state.setError
  }))

  const loadProjects = useCallback(async (signal?: AbortSignal) => {
    if (!signal?.aborted) {
      setLoading(true)
      setError(null)
    }

    try {
      const data = await fetchProjects(signal ? { signal } : undefined)
      if (!signal?.aborted) {
        setProjects(data)
      }
    } catch (err) {
      if (signal?.aborted || isAbortError(err)) {
        return
      }
      console.error('Failed to load projects', err)
      setError('Unable to load projects. Try again in a moment.')
    } finally {
      if (!signal?.aborted) {
        setLoading(false)
      }
    }
  }, [setError, setLoading, setProjects])

  useEffect(() => {
    const controller = new AbortController()
    void loadProjects(controller.signal)
    return () => {
      controller.abort()
    }
  }, [loadProjects])

  const activeFunnels = funnels.filter((funnel) => funnel.status === 'active').length
  const draftFunnels = funnels.filter((funnel) => funnel.status === 'draft').length
  const totalSteps = funnels.reduce((sum, funnel) => sum + funnel.steps.length, 0)
  const averageSteps = funnels.length > 0 ? (totalSteps / funnels.length).toFixed(1) : '0.0'

  const latestUpdatedAt = funnels.reduce<string | null>((latest, funnel) => {
    const candidate = funnel.updatedAt
    if (!candidate) {
      return latest
    }

    const candidateDate = new Date(candidate)
    if (Number.isNaN(candidateDate.getTime())) {
      return latest
    }

    if (!latest) {
      return candidate
    }

    const latestDate = new Date(latest)
    return candidateDate > latestDate ? candidate : latest
  }, null)

  const lastUpdated = formatDateTime(latestUpdatedAt, {
    dateStyle: 'medium',
    timeStyle: 'short'
  })

  const [isProjectModalOpen, setProjectModalOpen] = useState(false)
  const [projectModalMode, setProjectModalMode] = useState<'create' | 'edit'>('create')
  const [projectDraft, setProjectDraft] = useState({ name: '', description: '' })
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null)
  const [projectModalError, setProjectModalError] = useState<string | null>(null)
  const [isSavingProject, setIsSavingProject] = useState(false)

  const orderedProjects = useMemo(() => {
    return [...projects].sort((a, b) => {
      const first = new Date(a.updatedAt).getTime()
      const second = new Date(b.updatedAt).getTime()
      return second - first
    })
  }, [projects])

  const openCreateProject = () => {
    setProjectDraft({ name: '', description: '' })
    setActiveProjectId(null)
    setProjectModalMode('create')
    setProjectModalError(null)
    setProjectModalOpen(true)
  }

  const openEditProject = (project: Project) => {
    setProjectDraft({ name: project.name, description: project.description ?? '' })
    setActiveProjectId(project.id)
    setProjectModalMode('edit')
    setProjectModalError(null)
    setProjectModalOpen(true)
  }

  const closeProjectModal = () => {
    setProjectModalOpen(false)
    setProjectModalError(null)
    setIsSavingProject(false)
    setProjectDraft({ name: '', description: '' })
    setActiveProjectId(null)
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
      if (projectModalMode === 'create') {
        await createProject({
          name: trimmedName,
          description: trimmedDescription !== '' ? trimmedDescription : undefined
        })
        toast.success('Project created')
      } else if (activeProjectId) {
        await updateProject(activeProjectId, {
          name: trimmedName,
          description: trimmedDescription !== '' ? trimmedDescription : undefined
        })
        toast.success('Project updated')
      }

      await loadProjects()
      closeProjectModal()
    } catch (err) {
      console.error('Failed to save project', err)
      setProjectModalError('Unable to save project. Try again in a moment.')
      toast.error('Failed to save project')
    } finally {
      setIsSavingProject(false)
    }
  }

  const stats = [
    { label: 'Projects', value: projects.length.toString(), icon: Folder, color: 'text-indigo-600' },
    { label: 'Total Funnels', value: funnels.length.toString(), icon: Target, color: 'text-blue-600' },
    { label: 'Active Funnels', value: activeFunnels.toString(), icon: Users, color: 'text-green-600' },
    { label: 'Draft Funnels', value: draftFunnels.toString(), icon: TrendingUp, color: 'text-purple-600' },
    { label: 'Avg. Steps', value: averageSteps, icon: Clock, color: 'text-orange-600' },
  ]

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6 sm:mb-8">
        <h2 className="text-2xl font-bold text-gray-900 sm:text-3xl mb-2">Dashboard</h2>
        <p className="text-gray-600 text-sm sm:text-base">Manage your funnels and track performance</p>
      </div>

      {error && (
        <div className="mb-6 rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-800">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            <p className="text-sm font-medium">{error}</p>
          </div>
        </div>
      )}

      {isLoading && (
        <div className="mb-6 flex items-center gap-3 rounded-lg border border-gray-200 bg-white p-4 text-gray-600">
          <RefreshCw className="h-4 w-4 animate-spin" />
          <span className="text-sm font-medium">Loading projects…</span>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-4 mb-6 sm:mb-8">
        {stats.map((stat) => (
          <div key={stat.label} className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <stat.icon className={`w-8 h-8 ${stat.color}`} />
            </div>
            <p className="text-2xl font-bold text-gray-900 mb-1">{stat.value}</p>
            <p className="text-sm text-gray-600">{stat.label}</p>
          </div>
        ))}
      </div>

      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="text-lg font-bold text-gray-900 sm:text-xl">Your Projects</h3>
          <p className="text-sm text-gray-500">Last updated: {lastUpdated}</p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <button
            type="button"
            onClick={openCreateProject}
            className="btn btn-outline"
          >
            <Folder className="w-4 h-4 mr-2" />
            New Project
          </button>
          <button
            type="button"
            onClick={() => navigate('/builder')}
            className="btn btn-primary"
          >
            <Plus className="w-4 h-4 mr-2" />
            Create Funnel
          </button>
        </div>
      </div>

      {orderedProjects.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white p-12 text-center">
          <h3 className="text-lg font-semibold text-gray-900">Organize funnels by creating a project</h3>
          <p className="mt-2 text-sm text-gray-600">Projects help keep related funnels together for easier management.</p>
          <button
            type="button"
            onClick={openCreateProject}
            className="btn btn-primary mt-4"
          >
            <Folder className="w-4 h-4 mr-2" />
            Create Project
          </button>
        </div>
      ) : (
        <div className="space-y-6">
          {orderedProjects.map((project) => {
            const projectFunnels = project.funnels ?? []
            const funnelCountLabel = projectFunnels.length === 1 ? 'funnel' : 'funnels'

            return (
              <div key={project.id} className="card p-6">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <h4 className="text-lg font-semibold text-gray-900">{project.name}</h4>
                    {project.description && (
                      <p className="mt-1 text-sm text-gray-600">{project.description}</p>
                    )}
                  </div>
                  <button
                    type="button"
                    onClick={() => openEditProject(project)}
                    className="btn btn-outline self-start"
                  >
                    <Pencil className="mr-2 h-4 w-4" />
                    Edit Project
                  </button>
                </div>

                <div className="mt-4 flex flex-wrap items-center gap-4 text-sm text-gray-500">
                  <span>{projectFunnels.length} {funnelCountLabel}</span>
                  <span>Updated {formatDateTime(project.updatedAt, { dateStyle: 'medium' })}</span>
                </div>

                {projectFunnels.length === 0 ? (
                  <div className="mt-4 rounded-md border border-dashed border-gray-300 bg-white p-6 text-center text-sm text-gray-600">
                    No funnels in this project yet. Use the builder to create one.
                  </div>
                ) : (
                  <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
                    {projectFunnels.map((funnel) => (
                      <button
                        key={funnel.id}
                        type="button"
                        onClick={() => navigate(`/builder/${funnel.id}`)}
                        className="text-left rounded-lg border border-gray-200 bg-white p-5 transition hover:shadow"
                      >
                        <div className="flex items-start justify-between">
                          <div>
                            <p className="text-base font-semibold text-gray-900">{funnel.name}</p>
                            <p className="mt-1 text-sm text-gray-600 truncate">{funnel.description ?? 'No description provided.'}</p>
                          </div>
                          <span className={`rounded-full px-2 py-1 text-xs ${
                            funnel.status === 'active' ? 'bg-green-100 text-green-700' :
                            funnel.status === 'draft' ? 'bg-yellow-100 text-yellow-700' :
                            'bg-gray-100 text-gray-700'
                          }`}>
                            {funnel.status}
                          </span>
                        </div>
                        <div className="mt-4 flex justify-between text-sm text-gray-500">
                          <span>{funnel.steps.length} steps</span>
                          <span>Updated {formatDateTime(funnel.updatedAt, { dateStyle: 'medium' })}</span>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {isProjectModalOpen && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 px-4">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <div className="mb-5">
              <h3 className="text-lg font-semibold text-gray-900">
                {projectModalMode === 'create' ? 'Create Project' : 'Edit Project'}
              </h3>
              <p className="text-sm text-gray-600">
                {projectModalMode === 'create'
                  ? 'Group related funnels together for better organization.'
                  : 'Update the project details to keep your workspace organized.'}
              </p>
            </div>
            <form onSubmit={handleProjectSubmit} className="space-y-4">
              <div>
                <label htmlFor="project-name" className="block text-sm font-medium text-gray-700">
                  Project Name
                </label>
                <input
                  id="project-name"
                  name="project-name"
                  type="text"
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-200"
                  value={projectDraft.name}
                  onChange={(event) => setProjectDraft((prev) => ({ ...prev, name: event.target.value }))}
                  autoFocus
                  disabled={isSavingProject}
                />
              </div>
              <div>
                <label htmlFor="project-description" className="block text-sm font-medium text-gray-700">
                  Description <span className="text-gray-400">(optional)</span>
                </label>
                <textarea
                  id="project-description"
                  name="project-description"
                  rows={3}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-200"
                  value={projectDraft.description}
                  onChange={(event) => setProjectDraft((prev) => ({ ...prev, description: event.target.value }))}
                  disabled={isSavingProject}
                />
              </div>

              {projectModalError && (
                <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                  {projectModalError}
                </div>
              )}

              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  className="btn btn-outline"
                  onClick={closeProjectModal}
                  disabled={isSavingProject}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={isSavingProject}
                >
                  {isSavingProject ? 'Saving…' : projectModalMode === 'create' ? 'Create Project' : 'Save Changes'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Dashboard
