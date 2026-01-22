/**
 * Avatar Provider - Dependency injection for avatar components.
 * Allows easy swapping of avatar implementations.
 */

import React, { createContext, useContext, useMemo } from 'react'
import type { AvatarConfig, AvatarRegistry, AvatarProps } from '@/types/skilltree'
import { GeometricAvatar } from './avatars/GeometricAvatar'

// Avatar registry - add new avatars here
const AVATAR_REGISTRY: AvatarRegistry = {
  geometric: {
    Component: GeometricAvatar,
    displayName: 'Geometric Avatar',
    description: 'Abstract geometric avatar built with Three.js primitives',
  },
  // Future avatars can be added here:
  // mixamo: { Component: MixamoAvatar, preloadAssets: () => loadModel(), displayName: 'Animated' },
  // rive: { Component: RiveAvatar, displayName: '2.5D Character' },
}

// Default avatar
const DEFAULT_AVATAR = 'geometric'

// Context
const AvatarContext = createContext<AvatarConfig | null>(null)

interface AvatarProviderProps {
  avatar?: string
  children: React.ReactNode
}

/**
 * Provider component for avatar dependency injection.
 */
export function AvatarProvider({ avatar = DEFAULT_AVATAR, children }: AvatarProviderProps) {
  const config = useMemo((): AvatarConfig => {
    const selectedConfig = AVATAR_REGISTRY[avatar]
    if (!selectedConfig) {
      console.warn(`Avatar "${avatar}" not found, falling back to ${DEFAULT_AVATAR}`)
      return AVATAR_REGISTRY[DEFAULT_AVATAR] as AvatarConfig
    }
    return selectedConfig
  }, [avatar])

  return <AvatarContext.Provider value={config}>{children}</AvatarContext.Provider>
}

/**
 * Hook to access the current avatar configuration.
 */
export function useAvatar(): AvatarConfig {
  const context = useContext(AvatarContext)
  if (!context) {
    throw new Error('useAvatar must be used within an AvatarProvider')
  }
  return context
}

/**
 * Hook to get the avatar component.
 */
export function useAvatarComponent(): React.ComponentType<AvatarProps> {
  const { Component } = useAvatar()
  return Component
}

/**
 * Get list of available avatars.
 */
export function getAvailableAvatars(): Array<{ id: string; displayName: string; description?: string }> {
  return Object.entries(AVATAR_REGISTRY).map(([id, config]) => ({
    id,
    displayName: config.displayName,
    description: config.description,
  }))
}

/**
 * Register a new avatar at runtime.
 */
export function registerAvatar(id: string, config: AvatarConfig): void {
  if (AVATAR_REGISTRY[id]) {
    console.warn(`Avatar "${id}" already exists and will be overwritten`)
  }
  AVATAR_REGISTRY[id] = config
}

export { AVATAR_REGISTRY }
