/**
 * App.tsx - Main application entry point.
 *
 * Provides:
 * - Error boundaries for graceful error handling
 * - Theme context for light/dark mode support
 *
 * Note: QueryClientProvider is already configured in main.tsx.
 * The actual layout and state management is handled by PromptManagerLayout.
 */

import { ErrorBoundary } from './components/ErrorBoundary'
import { ThemeProvider } from './hooks/use-theme'
import { PromptManagerLayout } from './components/layout/PromptManagerLayout'

export default function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <PromptManagerLayout />
      </ThemeProvider>
    </ErrorBoundary>
  )
}
