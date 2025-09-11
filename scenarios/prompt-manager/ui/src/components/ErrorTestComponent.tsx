import { useState } from 'react'
import { Button } from './ui/button'

// This component is only for testing error boundaries
// It can be removed once error boundary testing is complete
export function ErrorTestComponent() {
  const [shouldThrow, setShouldThrow] = useState(false)

  if (shouldThrow) {
    throw new Error('Test error for error boundary demonstration')
  }

  return (
    <div className="fixed top-4 left-4 z-50 bg-red-100 dark:bg-red-900 p-3 rounded-lg border border-red-300 dark:border-red-700">
      <p className="text-sm text-red-800 dark:text-red-200 mb-2">
        Error Boundary Test
      </p>
      <Button
        size="sm"
        variant="destructive"
        onClick={() => setShouldThrow(true)}
      >
        Trigger Error
      </Button>
    </div>
  )
}