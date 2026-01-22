/**
 * App.tsx - Main application entry point.
 *
 * Simplified wrapper that provides:
 * - Theme context
 * - Error boundary
 * - React Query provider
 *
 * The actual layout and state management is handled by PromptManagerLayout.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ErrorBoundary } from './components/ErrorBoundary'
import { PromptManagerLayout } from './components/layout/PromptManagerLayout'

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5000, // 5 seconds
      refetchOnWindowFocus: false,
    },
  },
})

export default function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <ErrorBoundary>
          <PromptManagerLayout />
        </ErrorBoundary>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}
