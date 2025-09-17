import clsx from 'clsx'

interface BadgeProps {
  children: React.ReactNode
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'
  size?: 'sm' | 'md'
  className?: string
}

export function Badge({ 
  children, 
  variant = 'default', 
  size = 'md',
  className 
}: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center font-medium rounded-full',
        size === 'sm' && 'px-2 py-0.5 text-xs',
        size === 'md' && 'px-2.5 py-1 text-sm',
        variant === 'default' && 'bg-dark-100 text-dark-700',
        variant === 'primary' && 'bg-primary-100 text-primary-700',
        variant === 'success' && 'bg-success-50 text-success-700',
        variant === 'warning' && 'bg-warning-50 text-warning-700',
        variant === 'danger' && 'bg-danger-50 text-danger-700',
        variant === 'info' && 'bg-blue-50 text-blue-700',
        className
      )}
    >
      {children}
    </span>
  )
}