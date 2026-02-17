/**
 * Overlay store for managing 2D overlays in 3D space.
 * Handles speech bubbles, thinking states, and other visual indicators.
 */

import { create } from 'zustand'

/** Speech bubble message */
export interface SpeechBubble {
  id: string
  agentId: string
  text: string
  /** Duration in ms before auto-hide (0 = manual dismiss) */
  duration: number
  /** Timestamp when created */
  createdAt: number
  /** Whether this is a temporary message */
  temporary: boolean
}

/** Thinking indicator state */
export interface ThinkingState {
  agentId: string
  isThinking: boolean
  /** Optional label like "Processing..." */
  label?: string
}

/** Name tag visibility configuration */
export interface NameTagConfig {
  /** Show name tags for all agents */
  showAll: boolean
  /** Show only on hover */
  showOnHover: boolean
  /** Specific agent IDs to always show */
  alwaysShowFor: string[]
  /** Specific agent IDs to never show */
  neverShowFor: string[]
}

interface OverlayState {
  /** Active speech bubbles */
  speechBubbles: SpeechBubble[]
  /** Agents currently in thinking state */
  thinkingStates: Partial<Record<string, ThinkingState>>
  /** Name tag configuration */
  nameTagConfig: NameTagConfig
  /** Global overlay visibility */
  overlaysVisible: boolean
}

interface OverlayActions {
  /** Show a speech bubble for a agent */
  showSpeechBubble: (agentId: string, text: string, duration?: number) => string
  /** Hide a specific speech bubble */
  hideSpeechBubble: (id: string) => void
  /** Hide all speech bubbles for a agent */
  hideAllSpeechBubbles: (agentId: string) => void
  /** Set thinking state for a agent */
  setThinking: (agentId: string, isThinking: boolean, label?: string) => void
  /** Clear thinking state for a agent */
  clearThinking: (agentId: string) => void
  /** Update name tag configuration */
  updateNameTagConfig: (config: Partial<NameTagConfig>) => void
  /** Toggle global overlay visibility */
  setOverlaysVisible: (visible: boolean) => void
  /** Check if name tag should be shown for a agent */
  shouldShowNameTag: (agentId: string, isHovered: boolean) => boolean
  /** Clean up expired speech bubbles */
  cleanupExpiredBubbles: () => void
  /** Reset all overlay state */
  reset: () => void
}

type OverlayStore = OverlayState & OverlayActions

const initialState: OverlayState = {
  speechBubbles: [],
  thinkingStates: {},
  nameTagConfig: {
    showAll: true,
    showOnHover: false,
    alwaysShowFor: [],
    neverShowFor: [],
  },
  overlaysVisible: true,
}

let bubbleIdCounter = 0

/**
 * Zustand store for overlay management
 */
export const useOverlayStore = create<OverlayStore>((set, get) => ({
  ...initialState,

  showSpeechBubble: (agentId, text, duration = 5000) => {
    const id = `bubble-${++bubbleIdCounter}`
    const bubble: SpeechBubble = {
      id,
      agentId,
      text,
      duration,
      createdAt: Date.now(),
      temporary: duration > 0,
    }

    set({ speechBubbles: [...get().speechBubbles, bubble] })

    // Auto-remove after duration
    if (duration > 0) {
      setTimeout(() => {
        get().hideSpeechBubble(id)
      }, duration)
    }

    return id
  },

  hideSpeechBubble: (id) => {
    set({
      speechBubbles: get().speechBubbles.filter((b) => b.id !== id),
    })
  },

  hideAllSpeechBubbles: (agentId) => {
    set({
      speechBubbles: get().speechBubbles.filter((b) => b.agentId !== agentId),
    })
  },

  setThinking: (agentId, isThinking, label) => {
    const existing = get().thinkingStates[agentId]
    if (
      existing &&
      existing.isThinking === isThinking &&
      existing.label === label
    ) {
      return
    }
    set({
      thinkingStates: {
        ...get().thinkingStates,
        [agentId]: { agentId, isThinking, label },
      },
    })
  },

  clearThinking: (agentId) => {
    const { thinkingStates } = get()
    const { [agentId]: _, ...rest } = thinkingStates
    void _
    set({ thinkingStates: rest })
  },

  updateNameTagConfig: (config) => {
    set({
      nameTagConfig: { ...get().nameTagConfig, ...config },
    })
  },

  setOverlaysVisible: (visible) => {
    set({ overlaysVisible: visible })
  },

  shouldShowNameTag: (agentId, isHovered) => {
    const { nameTagConfig, overlaysVisible } = get()

    if (!overlaysVisible) return false
    if (nameTagConfig.neverShowFor.includes(agentId)) return false
    if (nameTagConfig.alwaysShowFor.includes(agentId)) return true
    if (nameTagConfig.showOnHover) return isHovered
    return nameTagConfig.showAll
  },

  cleanupExpiredBubbles: () => {
    const now = Date.now()
    set({
      speechBubbles: get().speechBubbles.filter((b) => {
        if (!b.temporary) return true
        return now - b.createdAt < b.duration
      }),
    })
  },

  reset: () => set(initialState),
}))

/**
 * Selector for speech bubbles for a specific agent.
 * Returns empty array for invalid agentId to prevent crashes.
 */
export const selectAgentSpeechBubbles = (state: OverlayStore, agentId: string | null | undefined): SpeechBubble[] => {
  if (!agentId) {
    return []
  }
  return state.speechBubbles.filter((b) => b.agentId === agentId)
}

/**
 * Selector for thinking state of a specific agent
 */
export const selectAgentThinking = (state: OverlayStore, agentId: string) =>
  state.thinkingStates[agentId] ?? null
