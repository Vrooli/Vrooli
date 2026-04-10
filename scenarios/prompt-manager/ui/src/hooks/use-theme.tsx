/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useContext, useEffect, useState } from 'react'
import type { Theme } from '@/types'

interface ThemeContextType {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('theme') as Theme
      if (['light', 'dark', 'system'].includes(stored)) {
        return stored
      }
    }
    return 'system'
  })

  useEffect(() => {
    const root = window.document.documentElement
    
    const applyTheme = (theme: Theme) => {
      root.classList.remove('light', 'dark')
      
      if (theme === 'system') {
        const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
        root.classList.add(systemTheme)
      } else {
        root.classList.add(theme)
      }
    }

    applyTheme(theme)
    localStorage.setItem('theme', theme)

    // Listen for system theme changes when in system mode
    if (theme === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      const handleChange = () => applyTheme(theme)
      mediaQuery.addEventListener('change', handleChange)
      return () => mediaQuery.removeEventListener('change', handleChange)
    }
  }, [theme])

  const toggleTheme = () => {
    setTheme(prev => {
      const themes: Theme[] = ['light', 'dark', 'system']
      const currentIndex = themes.indexOf(prev)
      const nextIndex = (currentIndex + 1) % themes.length
      return themes[nextIndex] ?? 'system'
    })
  }

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}

/**
 * Hook to get the resolved theme (always 'light' or 'dark', never 'system').
 * Useful for components that need the actual theme value, like 3D renderers.
 */
export function useResolvedTheme(): 'light' | 'dark' {
  const { theme } = useTheme()
  const [resolved, setResolved] = useState<'light' | 'dark'>(() => {
    if (theme === 'system') {
      return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light'
    }
    return theme
  })

  useEffect(() => {
    if (theme !== 'system') {
      setResolved(theme)
      return
    }

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const handleChange = (e: MediaQueryListEvent) => {
      setResolved(e.matches ? 'dark' : 'light')
    }

    setResolved(mediaQuery.matches ? 'dark' : 'light')
    mediaQuery.addEventListener('change', handleChange)
    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [theme])

  return resolved
}