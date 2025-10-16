interface LoadingSpinnerProps {
  size?: 'small' | 'medium' | 'large'
  message?: string
}

function LoadingSpinner({ size = 'medium', message }: LoadingSpinnerProps) {
  const sizeClasses = {
    small: 'h-6 w-6 border-2',
    medium: 'h-10 w-10 border-[3px]',
    large: 'h-14 w-14 border-[4px]',
  } as const

  return (
    <div className="flex flex-col items-center justify-center gap-3 py-12 text-center">
      <div
        className={`animate-spin rounded-full border-border/60 border-t-primary ${sizeClasses[size]}`}
      />
      {message && (
        <p className="max-w-sm text-sm font-medium text-muted-foreground">{message}</p>
      )}
    </div>
  )
}

export default LoadingSpinner
