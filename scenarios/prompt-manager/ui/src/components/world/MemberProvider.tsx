/**
 * Member Provider - Dependency injection for member components.
 * Allows easy swapping of member implementations.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#dependency-injection-pattern
// DOC: docs/SEAMS.md#1-memberprovider-dependency-injection

/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useContext, useMemo } from 'react'
import type { MemberConfig, MemberRegistry, MemberProps } from '@/types/world'
import { GeometricMember } from './members/GeometricMember'

// Member registry - add new members here
const MEMBER_REGISTRY: MemberRegistry = {
  geometric: {
    Component: GeometricMember,
    displayName: 'Geometric Member',
    description: 'Abstract geometric member built with Three.js primitives',
  },
  // Future members can be added here:
  // mixamo: { Component: MixamoMember, preloadAssets: () => loadModel(), displayName: 'Animated' },
  // rive: { Component: RiveMember, displayName: '2.5D Character' },
}

// Default member
const DEFAULT_MEMBER = 'geometric'

// Context
const MemberContext = createContext<MemberConfig | null>(null)

interface MemberProviderProps {
  member?: string
  children: React.ReactNode
}

/**
 * Provider component for member dependency injection.
 */
export function MemberProvider({ member = DEFAULT_MEMBER, children }: MemberProviderProps) {
  const config = useMemo((): MemberConfig => {
    const selectedConfig = MEMBER_REGISTRY[member]
    if (!selectedConfig) {
      console.warn(`Member "${member}" not found, falling back to ${DEFAULT_MEMBER}`)
      return MEMBER_REGISTRY[DEFAULT_MEMBER] as MemberConfig
    }
    return selectedConfig
  }, [member])

  return <MemberContext.Provider value={config}>{children}</MemberContext.Provider>
}

/**
 * Hook to access the current member configuration.
 */
export function useMember(): MemberConfig {
  const context = useContext(MemberContext)
  if (!context) {
    throw new Error('useMember must be used within a MemberProvider')
  }
  return context
}

/**
 * Hook to get the member component.
 */
export function useMemberComponent(): React.ComponentType<MemberProps> {
  const { Component } = useMember()
  return Component
}

/**
 * Get list of available members.
 */
export function getAvailableMembers(): Array<{ id: string; displayName: string; description?: string }> {
  return Object.entries(MEMBER_REGISTRY).map(([id, config]) => ({
    id,
    displayName: config.displayName,
    description: config.description,
  }))
}

/**
 * Register a new member at runtime.
 */
export function registerMember(id: string, config: MemberConfig): void {
  if (MEMBER_REGISTRY[id]) {
    console.warn(`Member "${id}" already exists and will be overwritten`)
  }
  MEMBER_REGISTRY[id] = config
}

export { MEMBER_REGISTRY }
