/**
 * App.tsx - Main application entry point.
 *
 * Provides:
 * - Error boundaries for graceful error handling
 * - Theme context for light/dark mode support
 *
 * Note: QueryClientProvider is already configured in main.tsx.
 * The actual layout and state management is handled by SkillManagerLayout.
 */

import { ErrorBoundary } from './components/ErrorBoundary'
import { ThemeProvider } from './hooks/use-theme'
import { SkillManagerLayout } from './components/layout/SkillManagerLayout'

export default function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <SkillManagerLayout />
      </ThemeProvider>
    </ErrorBoundary>
  )
}
