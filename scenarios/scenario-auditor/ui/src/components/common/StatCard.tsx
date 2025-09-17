import { LucideIcon, TrendingUp, TrendingDown, Minus, AlertTriangle, Info } from 'lucide-react'
import clsx from 'clsx'
import { useState } from 'react'

interface StatCardProps {
  title: string
  value: number | string | null
  unit?: string
  icon: LucideIcon
  trend?: 'up' | 'down' | 'stable'
  detail?: string
  color?: 'primary' | 'success' | 'warning' | 'danger'
  loading?: boolean
  warningMessage?: string
  hasWarning?: boolean
}

export function StatCard({ 
  title, 
  value, 
  unit, 
  icon: Icon, 
  trend, 
  detail,
  color = 'primary',
  loading,
  warningMessage,
  hasWarning
}: StatCardProps) {
  const [showTooltip, setShowTooltip] = useState(false)

  const getTrendIcon = () => {
    if (!trend) return null
    switch (trend) {
      case 'up': return <TrendingUp className="h-4 w-4 text-success-500" />
      case 'down': return <TrendingDown className="h-4 w-4 text-danger-500" />
      case 'stable': return <Minus className="h-4 w-4 text-dark-400" />
    }
  }

  const getColorClasses = () => {
    switch (color) {
      case 'success': return 'from-success-500 to-success-600'
      case 'warning': return 'from-warning-500 to-warning-600'
      case 'danger': return 'from-danger-500 to-danger-600'
      default: return 'from-primary-500 to-primary-600'
    }
  }

  if (loading) {
    return (
      <div className="bg-white rounded-xl p-6 shadow-sm border border-dark-200">
        <div className="space-y-3">
          <div className="h-4 bg-dark-200 rounded animate-pulse w-24" />
          <div className="h-8 bg-dark-200 rounded animate-pulse w-32" />
          <div className="h-3 bg-dark-200 rounded animate-pulse w-20" />
        </div>
      </div>
    )
  }

  // Display warning state if no scans have been performed
  const displayValue = value === null ? '—' : value
  const showWarning = hasWarning || value === null

  return (
    <div className="bg-white rounded-xl p-6 shadow-sm border border-dark-200 hover:shadow-md transition-shadow relative">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium text-dark-600">{title}</p>
            {showWarning && warningMessage && (
              <div className="relative">
                <button
                  onMouseEnter={() => setShowTooltip(true)}
                  onMouseLeave={() => setShowTooltip(false)}
                  className="text-warning-500 hover:text-warning-600 transition-colors"
                  aria-label="Warning information"
                >
                  <AlertTriangle className="h-4 w-4" />
                </button>
                {showTooltip && (
                  <div className="absolute z-10 left-0 top-6 w-64 p-3 bg-dark-900 text-white text-xs rounded-lg shadow-lg">
                    <div className="absolute -top-1 left-3 w-2 h-2 bg-dark-900 rotate-45"></div>
                    {warningMessage}
                  </div>
                )}
              </div>
            )}
          </div>
          <div className="mt-2 flex items-baseline gap-1">
            <span className={clsx(
              "text-3xl font-bold",
              showWarning ? "text-warning-600" : "text-dark-900"
            )}>
              {displayValue}
            </span>
            {unit && value !== null && (
              <span className="text-lg font-medium text-dark-500">{unit}</span>
            )}
          </div>
          {detail && (
            <p className="mt-1 text-xs text-dark-500">{detail}</p>
          )}
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className={clsx(
            'flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br text-white',
            showWarning ? 'from-warning-500 to-warning-600' : getColorClasses()
          )}>
            <Icon className="h-5 w-5" />
          </div>
          {!showWarning && getTrendIcon()}
        </div>
      </div>
    </div>
  )
}