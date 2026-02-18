/**
 * useMediaQuery - Hook for responsive design using media queries.
 *
 * Features:
 * - SSR-safe (returns false during SSR)
 * - Updates on window resize
 * - Cleans up event listeners on unmount
 */

import { useState, useEffect } from 'react'

export const BREAKPOINT_QUERIES = {
  compactHeader: '(max-width: 389px)',
  mobile: '(max-width: 768px)',
  tablet: '(min-width: 769px) and (max-width: 1024px)',
  desktop: '(min-width: 1025px)',
} as const

/**
 * Custom hook that tracks whether a CSS media query matches.
 * @param query - CSS media query string (e.g., '(max-width: 768px)')
 * @returns Boolean indicating if the media query matches
 */
export function useMediaQuery(query: string): boolean {
  // Default to false for SSR
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(query).matches
  })

  useEffect(() => {
    if (typeof window === 'undefined') return

    const mediaQuery = window.matchMedia(query)

    // Set initial value
    setMatches(mediaQuery.matches)

    // Handler for media query changes
    const handler = (event: MediaQueryListEvent) => {
      setMatches(event.matches)
    }

    // Add listener (modern browsers)
    mediaQuery.addEventListener('change', handler)

    // Cleanup
    return () => {
      mediaQuery.removeEventListener('change', handler)
    }
  }, [query])

  return matches
}

/**
 * Common breakpoint hooks for convenience.
 */
export function useIsMobile(): boolean {
  return useMediaQuery(BREAKPOINT_QUERIES.mobile)
}

export function useIsTablet(): boolean {
  return useMediaQuery(BREAKPOINT_QUERIES.tablet)
}

export function useIsDesktop(): boolean {
  return useMediaQuery(BREAKPOINT_QUERIES.desktop)
}

export function useIsCompactHeader(): boolean {
  return useMediaQuery(BREAKPOINT_QUERIES.compactHeader)
}
