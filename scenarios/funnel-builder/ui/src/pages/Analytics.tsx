import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell
} from 'recharts'
import {
  TrendingUp,
  Users,
  Target,
  Clock,
  ArrowUp,
  ArrowDown,
  Minus,
  Loader2,
  AlertTriangle,
  RefreshCw
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { useFunnelStore } from '../store/useFunnelStore'
import { fetchFunnelAnalytics, fetchFunnels } from '../services/funnels'
import type { FunnelAnalytics, DailyStat } from '../types'

const colorPalette = ['#0ea5e9', '#8b5cf6', '#10b981', '#f59e0b', '#ec4899', '#6366f1', '#14b8a6', '#f97316']

type TrendDirection = 'up' | 'down' | 'neutral'

interface StatCard {
  label: string
  value: string
  changeLabel: string
  trend: TrendDirection
  icon: LucideIcon
}

const formatDuration = (seconds?: number | null): string => {
  if (!seconds || Number.isNaN(seconds) || seconds <= 0) {
    return '—'
  }

  const totalSeconds = Math.round(seconds)
  const minutes = Math.floor(totalSeconds / 60)
  const remainingSeconds = totalSeconds % 60

  if (minutes === 0) {
    return `${remainingSeconds}s`
  }

  return `${minutes}m ${remainingSeconds.toString().padStart(2, '0')}s`
}

const formatNumber = (value: number): string => value.toLocaleString()

const formatPercentageLabel = (change: number | null): string => {
  if (change === null || Number.isNaN(change)) {
    return '—'
  }

  if (Math.abs(change) < 0.05) {
    return '0%'
  }

  const rounded = Math.round(change * 10) / 10
  return `${rounded > 0 ? '+' : ''}${rounded.toFixed(1)}%`
}

const formatPointChangeLabel = (change: number | null): string => {
  if (change === null || Number.isNaN(change)) {
    return '—'
  }

  if (Math.abs(change) < 0.05) {
    return '0 pts'
  }

  const rounded = Math.round(change * 10) / 10
  return `${rounded > 0 ? '+' : ''}${rounded.toFixed(1)} pts`
}

const computeAggregateChange = (stats: DailyStat[], key: 'views' | 'leads' | 'conversions'): {
  change: number | null
  trend: TrendDirection
} => {
  if (!stats.length) {
    return { change: null, trend: 'neutral' }
  }

  const sorted = [...stats].sort((a, b) => a.date.localeCompare(b.date))
  const currentWindow = sorted.slice(-7)
  const previousWindow = sorted.slice(-14, -7)

  const currentTotal = currentWindow.reduce((acc, entry) => acc + entry[key], 0)
  const previousTotal = previousWindow.reduce((acc, entry) => acc + entry[key], 0)

  if (!previousWindow.length || previousTotal === 0) {
    if (currentTotal === 0) {
      return { change: 0, trend: 'neutral' }
    }
    return { change: null, trend: currentTotal > 0 ? 'up' : 'neutral' }
  }

  const delta = ((currentTotal - previousTotal) / previousTotal) * 100
  const rounded = Math.round(delta * 10) / 10

  if (Math.abs(rounded) < 0.05) {
    return { change: 0, trend: 'neutral' }
  }

  return { change: rounded, trend: rounded > 0 ? 'up' : 'down' }
}

const computeConversionTrend = (stats: DailyStat[]): {
  change: number | null
  trend: TrendDirection
} => {
  if (!stats.length) {
    return { change: null, trend: 'neutral' }
  }

  const sorted = [...stats].sort((a, b) => a.date.localeCompare(b.date))
  const currentWindow = sorted.slice(-7)
  const previousWindow = sorted.slice(-14, -7)

  const currentViews = currentWindow.reduce((acc, entry) => acc + entry.views, 0)
  const currentConversions = currentWindow.reduce((acc, entry) => acc + entry.conversions, 0)
  const currentRate = currentViews > 0 ? (currentConversions / currentViews) * 100 : 0

  if (!previousWindow.length) {
    return { change: null, trend: currentRate > 0 ? 'up' : 'neutral' }
  }

  const previousViews = previousWindow.reduce((acc, entry) => acc + entry.views, 0)
  const previousConversions = previousWindow.reduce((acc, entry) => acc + entry.conversions, 0)
  const previousRate = previousViews > 0 ? (previousConversions / previousViews) * 100 : 0

  if (previousViews === 0) {
    return { change: null, trend: currentRate > 0 ? 'up' : 'neutral' }
  }

  const diff = currentRate - previousRate
  const rounded = Math.round(diff * 10) / 10

  if (Math.abs(rounded) < 0.05) {
    return { change: 0, trend: 'neutral' }
  }

  return { change: rounded, trend: rounded > 0 ? 'up' : 'down' }
}

const formatDailyLabel = (isoDate: string): string => {
  const date = new Date(`${isoDate}T00:00:00Z`)
  if (Number.isNaN(date.getTime())) {
    return isoDate
  }
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

const Analytics = () => {
  const navigate = useNavigate()
  const params = useParams<{ id?: string }>()
  const routeFunnelId = params.id

  const { funnels, setFunnels } = useFunnelStore((state) => ({
    funnels: state.funnels,
    setFunnels: state.setFunnels
  }))

  const [selectedFunnelId, setSelectedFunnelId] = useState<string | undefined>(routeFunnelId)
  const [funnelsLoading, setFunnelsLoading] = useState(false)
  const [analyticsLoading, setAnalyticsLoading] = useState(false)
  const [funnelsError, setFunnelsError] = useState<string | null>(null)
  const [analyticsError, setAnalyticsError] = useState<string | null>(null)
  const [analytics, setAnalytics] = useState<FunnelAnalytics | null>(null)
  const [refreshCounter, setRefreshCounter] = useState(0)

  useEffect(() => {
    if (routeFunnelId) {
      setSelectedFunnelId(routeFunnelId)
    }
  }, [routeFunnelId])

  useEffect(() => {
    let cancelled = false
    const controller = new AbortController()

    const loadFunnels = async () => {
      setFunnelsLoading(true)
      setFunnelsError(null)
      try {
        const data = await fetchFunnels({ signal: controller.signal })
        if (cancelled) {
          return
        }

        setFunnels(data)

        if (!data.length) {
          setSelectedFunnelId(undefined)
          return
        }

        const targetId = routeFunnelId && data.some((funnel) => funnel.id === routeFunnelId)
          ? routeFunnelId
          : data[0].id

        if (!routeFunnelId || targetId !== routeFunnelId) {
          navigate(targetId ? `/analytics/${targetId}` : '/analytics', { replace: true })
        }
        setSelectedFunnelId((current) => current ?? targetId)
      } catch (error) {
        if (cancelled) {
          return
        }
        console.error('Failed to load funnels', error)
        setFunnelsError('Unable to load funnels. Please try again.')
      } finally {
        if (!cancelled) {
          setFunnelsLoading(false)
        }
      }
    }

    loadFunnels()

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [navigate, routeFunnelId, setFunnels])

  useEffect(() => {
    if (!selectedFunnelId) {
      setAnalytics(null)
      return
    }

    let cancelled = false
    const controller = new AbortController()

    const loadAnalytics = async () => {
      setAnalyticsLoading(true)
      setAnalyticsError(null)
      try {
        const data = await fetchFunnelAnalytics(selectedFunnelId, { signal: controller.signal })
        if (!cancelled) {
          setAnalytics(data)
        }
      } catch (error) {
        if (cancelled) {
          return
        }
        console.error('Failed to load analytics', error)
        setAnalyticsError('Unable to load analytics for this funnel.')
      } finally {
        if (!cancelled) {
          setAnalyticsLoading(false)
        }
      }
    }

    loadAnalytics()

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [selectedFunnelId, refreshCounter])

  const selectedFunnel = useMemo(() => {
    if (!selectedFunnelId) {
      return undefined
    }
    return funnels.find((funnel) => funnel.id === selectedFunnelId)
  }, [funnels, selectedFunnelId])

  const viewTrend = useMemo(() => computeAggregateChange(analytics?.dailyStats ?? [], 'views'), [analytics])
  const leadTrend = useMemo(() => computeAggregateChange(analytics?.dailyStats ?? [], 'leads'), [analytics])
  const conversionTrend = useMemo(() => computeConversionTrend(analytics?.dailyStats ?? []), [analytics])

  const stats: StatCard[] = useMemo(() => {
    if (!analytics) {
      return [
        { label: 'Total Views', value: '—', changeLabel: '—', trend: 'neutral', icon: Target },
        { label: 'Total Leads', value: '—', changeLabel: '—', trend: 'neutral', icon: Users },
        { label: 'Conversion Rate', value: '—', changeLabel: '—', trend: 'neutral', icon: TrendingUp },
        { label: 'Avg. Time', value: '—', changeLabel: '—', trend: 'neutral', icon: Clock }
      ]
    }

    const conversionRate = Math.round((analytics.conversionRate ?? 0) * 10) / 10
    const captureCount = analytics.capturedLeads ?? analytics.completedLeads

    return [
      {
        label: 'Total Views',
        value: formatNumber(analytics.totalViews ?? 0),
        changeLabel: formatPercentageLabel(viewTrend.change),
        trend: viewTrend.trend,
        icon: Target
      },
      {
        label: 'Total Leads',
        value: formatNumber(captureCount ?? 0),
        changeLabel: formatPercentageLabel(leadTrend.change),
        trend: leadTrend.trend,
        icon: Users
      },
      {
        label: 'Conversion Rate',
        value: `${conversionRate.toFixed(1)}%`,
        changeLabel: formatPointChangeLabel(conversionTrend.change),
        trend: conversionTrend.trend,
        icon: TrendingUp
      },
      {
        label: 'Avg. Time',
        value: formatDuration(analytics.averageTime),
        changeLabel: '—',
        trend: 'neutral',
        icon: Clock
      }
    ]
  }, [analytics, conversionTrend.change, conversionTrend.trend, leadTrend.change, leadTrend.trend, viewTrend.change, viewTrend.trend])

  const dailyPerformanceData = useMemo(() => {
    if (!analytics) {
      return []
    }

    return analytics.dailyStats.map((entry) => ({
      date: formatDailyLabel(entry.date),
      views: entry.views,
      leads: entry.leads,
      conversions: entry.conversions
    }))
  }, [analytics])

  const hasDailyData = dailyPerformanceData.some((entry) => entry.views || entry.leads || entry.conversions)

  const dropOffChartData = useMemo(() => {
    if (!analytics) {
      return []
    }

    const totalVisitors = Math.max(analytics.totalLeads ?? 0, analytics.totalViews ?? 0)
    let previousVisitors = totalVisitors

    return analytics.dropOffPoints.map((point) => {
      const visitors = point.visitors || previousVisitors
      const completions = point.responses || 0
      const dropOff = Math.max(visitors - completions, 0)
      previousVisitors = completions

      return {
        step: point.stepTitle || 'Step',
        visitors,
        dropOff
      }
    })
  }, [analytics])

  const hasDropOffData = dropOffChartData.some((entry) => entry.visitors || entry.dropOff)

  const trafficSourceData = useMemo(() => {
    if (!analytics?.trafficSources?.length) {
      return []
    }

    return analytics.trafficSources.map((source, index) => ({
      name: source.source,
      value: source.count,
      percentage: source.percentage,
      color: colorPalette[index % colorPalette.length]
    }))
  }, [analytics])

  const stepHighlights = useMemo(() => {
    if (!analytics?.dropOffPoints?.length) {
      return []
    }

    return [...analytics.dropOffPoints]
      .map((point, index) => {
        const visitors = point.visitors || 0
        const completionRate = visitors > 0 ? (point.responses / visitors) * 100 : 0
        return {
          step: point.stepTitle || `Step ${index + 1}`,
          visitors,
          completionRate
        }
      })
      .sort((a, b) => b.completionRate - a.completionRate)
      .slice(0, 4)
  }, [analytics])

  const handleFunnelChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value || undefined
    setSelectedFunnelId(value)
    if (value) {
      navigate(`/analytics/${value}`, { replace: true })
      setRefreshCounter((counter) => counter + 1)
    }
  }

  const handleRefresh = () => {
    if (selectedFunnelId) {
      setRefreshCounter((counter) => counter + 1)
    }
  }

  return (
    <div className="p-8">
      <div className="mb-8 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="text-3xl font-bold text-gray-900 mb-2">Funnel Analytics</h2>
          <p className="text-gray-600">Track performance and optimize conversions</p>
        </div>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <div className="flex items-center gap-2">
            <label htmlFor="funnel-select" className="text-sm font-medium text-gray-600">
              Funnel
            </label>
            <select
              id="funnel-select"
              className="rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              value={selectedFunnelId ?? ''}
              onChange={handleFunnelChange}
              disabled={funnelsLoading || !funnels.length}
            >
              {funnels.length === 0 ? (
                <option value="">No funnels available</option>
              ) : (
                funnels.map((funnel) => (
                  <option key={funnel.id} value={funnel.id}>
                    {funnel.name}
                  </option>
                ))
              )}
            </select>
          </div>
          <button
            type="button"
            onClick={handleRefresh}
            disabled={!selectedFunnelId || analyticsLoading}
            className="inline-flex items-center gap-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {analyticsLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </button>
        </div>
      </div>

      {funnelsError && (
        <div className="mb-6 flex items-start gap-3 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>{funnelsError}</span>
        </div>
      )}

      {analyticsError && (
        <div className="mb-6 flex items-start gap-3 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>{analyticsError}</span>
        </div>
      )}

      {analyticsLoading && (
        <div className="mb-6 flex items-center gap-2 text-sm text-gray-600">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Loading analytics…</span>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {stats.map((stat) => {
          const trendColor = stat.trend === 'up' ? 'text-green-600' : stat.trend === 'down' ? 'text-red-600' : 'text-gray-500'
          const TrendIcon = stat.trend === 'up' ? ArrowUp : stat.trend === 'down' ? ArrowDown : Minus

          return (
            <div key={stat.label} className="card p-6">
              <div className="flex items-center justify-between mb-4">
                <stat.icon className="w-8 h-8 text-primary-500" />
                <div className={`flex items-center gap-1 text-sm ${trendColor}`}>
                  <TrendIcon className="h-4 w-4" />
                  {stat.changeLabel}
                </div>
              </div>
              <p className="text-2xl font-bold text-gray-900 mb-1">{stat.value}</p>
              <p className="text-sm text-gray-600">{stat.label}</p>
            </div>
          )
        })}
      </div>

      {!analyticsLoading && analytics && analytics.totalViews === 0 && (
        <div className="mb-8 rounded-md border border-dashed border-gray-300 bg-gray-50 p-6 text-center text-sm text-gray-600">
          No analytics data captured yet. Share your funnel to start collecting insights.
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Daily Performance</h3>
          {hasDailyData ? (
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={dailyPerformanceData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="date" stroke="#666" />
                <YAxis stroke="#666" />
                <Tooltip />
                <Line type="monotone" dataKey="views" stroke="#0ea5e9" strokeWidth={2} dot={{ fill: '#0ea5e9', r: 3 }} />
                <Line type="monotone" dataKey="leads" stroke="#10b981" strokeWidth={2} dot={{ fill: '#10b981', r: 3 }} />
                <Line type="monotone" dataKey="conversions" stroke="#8b5cf6" strokeWidth={2} dot={{ fill: '#8b5cf6', r: 3 }} />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex h-[300px] items-center justify-center text-sm text-gray-500">
              Not enough data to display daily performance yet.
            </div>
          )}
        </div>

        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Funnel Drop-off</h3>
          {hasDropOffData ? (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={dropOffChartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="step" stroke="#666" />
                <YAxis stroke="#666" />
                <Tooltip />
                <Bar dataKey="visitors" fill="#0ea5e9" />
                <Bar dataKey="dropOff" fill="#ef4444" />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex h-[300px] items-center justify-center text-sm text-gray-500">
              Drop-off analytics will appear once visitors move through the funnel.
            </div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Traffic Sources</h3>
          {trafficSourceData.length ? (
            <>
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie data={trafficSourceData} cx="50%" cy="50%" innerRadius={60} outerRadius={80} dataKey="value" paddingAngle={5}>
                    {trafficSourceData.map((entry, index) => (
                      <Cell key={`traffic-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip formatter={(value: number, _label: string, props: any) => [`${value} visits`, props.payload.name]} />
                </PieChart>
              </ResponsiveContainer>
              <div className="mt-4 space-y-2">
                {trafficSourceData.map((source) => (
                  <div key={source.name} className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <span
                        className="block h-3 w-3 rounded-full"
                        style={{ backgroundColor: source.color }}
                      />
                      <span className="text-gray-600">{source.name}</span>
                    </div>
                    <span className="font-medium text-gray-900">
                      {`${(Math.round(source.percentage * 10) / 10).toFixed(1)}%`}
                    </span>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="flex h-[250px] items-center justify-center text-sm text-gray-500">
              Traffic source attribution appears here once leads identify their origin.
            </div>
          )}
        </div>

        <div className="lg:col-span-2 card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Top Performing Steps</h3>
          {stepHighlights.length ? (
            <div className="space-y-3">
              {stepHighlights.map((step, index) => (
                <div key={step.step} className="flex items-center justify-between rounded-lg bg-gray-50 p-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 text-sm font-semibold text-primary-600">
                      {index + 1}
                    </div>
                    <div>
                      <p className="font-medium text-gray-900">{step.step}</p>
                      <p className="text-xs text-gray-500">{formatNumber(step.visitors)} visitors</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold text-gray-900">{step.completionRate.toFixed(1)}%</p>
                    <p className="text-xs text-gray-500">completion rate</p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex h-[180px] items-center justify-center text-sm text-gray-500">
              Step performance insights will appear once leads begin progressing through the funnel.
            </div>
          )}
        </div>
      </div>

      {selectedFunnel && (
        <div className="mt-8 text-sm text-gray-500">
          Viewing analytics for <span className="font-medium text-gray-700">{selectedFunnel.name}</span> — updated{' '}
          {new Date(selectedFunnel.updatedAt).toLocaleString()}
        </div>
      )}
    </div>
  )
}

export default Analytics
