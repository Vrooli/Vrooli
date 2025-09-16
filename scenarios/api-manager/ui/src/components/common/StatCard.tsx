import { LucideIcon, TrendingUp, TrendingDown, Minus } from 'lucide-react'
import clsx from 'clsx'

interface StatCardProps {
  title: string
  value: number | string
  unit?: string
  icon: LucideIcon
  trend?: 'up' | 'down' | 'stable'
  detail?: string
  color?: 'primary' | 'success' | 'warning' | 'danger'
  loading?: boolean
}

export function StatCard({ 
  title, 
  value, 
  unit, 
  icon: Icon, 
  trend, 
  detail,
  color = 'primary',
  loading 
}: StatCardProps) {
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

  return (
    <div className="bg-white rounded-xl p-6 shadow-sm border border-dark-200 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-dark-600">{title}</p>
          <div className="mt-2 flex items-baseline gap-1">
            <span className="text-3xl font-bold text-dark-900">
              {value}
            </span>
            {unit && (
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
            getColorClasses()
          )}>
            <Icon className="h-5 w-5" />
          </div>
          {getTrendIcon()}
        </div>
      </div>
    </div>
  )
}