import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, TrendingUp, Users, Target, Clock, AlertTriangle, RefreshCw } from 'lucide-react'
import { useFunnelStore } from '../store/useFunnelStore'
import type { Funnel } from '../types'
import { fetchFunnels } from '../services/funnels'

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
  const { funnels, isLoading, error, setFunnels, setLoading, setError } = useFunnelStore((state) => ({
    funnels: state.funnels,
    isLoading: state.isLoading,
    error: state.error,
    setFunnels: state.setFunnels,
    setLoading: state.setLoading,
    setError: state.setError
  }))

  useEffect(() => {
    const controller = new AbortController()
    let cancelled = false

    const loadFunnels = async () => {
      setLoading(true)
      setError(null)
      try {
        const data: Funnel[] = await fetchFunnels({ signal: controller.signal })
        if (!cancelled) {
          setFunnels(data)
        }
      } catch (err) {
        if (!cancelled && !isAbortError(err)) {
          console.error('Failed to load funnels', err)
          setError('Unable to load funnels. Try again in a moment.')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    loadFunnels()

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [setError, setFunnels, setLoading])

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

  const stats = [
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
          <span className="text-sm font-medium">Loading funnels…</span>
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
          <h3 className="text-lg font-bold text-gray-900 sm:text-xl">Your Funnels</h3>
          <p className="text-sm text-gray-500">Last updated: {lastUpdated}</p>
        </div>
        <button
          onClick={() => navigate('/builder')}
          className="btn btn-primary"
        >
          <Plus className="w-4 h-4 mr-2" />
          Create Funnel
        </button>
      </div>

      {funnels.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white p-12 text-center">
          <h3 className="text-lg font-semibold text-gray-900">You haven’t created any funnels yet</h3>
          <p className="mt-2 text-sm text-gray-600">Use the builder to create your first conversion funnel.</p>
          <button
            onClick={() => navigate('/builder')}
            className="btn btn-primary mt-4"
          >
            <Plus className="w-4 h-4 mr-2" />
            Create Funnel
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-3">
          {funnels.map((funnel) => (
            <div
              key={funnel.id}
              className="card p-6 hover:shadow-lg transition-shadow cursor-pointer"
              onClick={() => navigate(`/builder/${funnel.id}`)}
            >
              <div className="flex justify-between items-start mb-4">
                <h4 className="text-lg font-semibold text-gray-900">{funnel.name}</h4>
                <span className={`px-2 py-1 text-xs rounded-full ${
                  funnel.status === 'active' ? 'bg-green-100 text-green-700' :
                  funnel.status === 'draft' ? 'bg-yellow-100 text-yellow-700' :
                  'bg-gray-100 text-gray-700'
                }`}>
                  {funnel.status}
                </span>
              </div>
              <p className="text-gray-600 text-sm mb-4">{funnel.description}</p>
              <div className="flex justify-between text-sm text-gray-500">
                <span>{funnel.steps.length} steps</span>
                <span>Updated {formatDateTime(funnel.updatedAt, { dateStyle: 'medium' })}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default Dashboard
