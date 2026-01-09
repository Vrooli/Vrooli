import { useEffect, useState } from 'react'

/**
 * Custom hook for detecting responsive layout breakpoints.
 * Monitors window width and provides compact layout flag.
 *
 * @param breakpoint - Max width in pixels for compact layout (default: 1120)
 */
export const useResponsiveLayout = (breakpoint: number = 1120) => {
  const [isCompactLayout, setIsCompactLayout] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined
    }

    const mediaQuery = window.matchMedia(`(max-width: ${breakpoint}px)`)

    const handleChange = (event: MediaQueryListEvent) => {
      setIsCompactLayout(event.matches)
    }

    // Set initial value
    setIsCompactLayout(mediaQuery.matches)

    // Use modern API if available, fall back to deprecated one
    if (typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', handleChange)
      return () => mediaQuery.removeEventListener('change', handleChange)
    }

    // Deprecated but still supported in some browsers
    mediaQuery.addListener(handleChange)
    return () => mediaQuery.removeListener(handleChange)
  }, [breakpoint])

  return { isCompactLayout }
}
