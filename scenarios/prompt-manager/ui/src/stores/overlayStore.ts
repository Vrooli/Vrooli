/**
 * Overlay store for managing 2D overlays in 3D space.
 * Handles speech bubbles, thinking states, and other visual indicators.
 */

import { create } from 'zustand'

/** Speech bubble message */
export interface SpeechBubble {
  id: string
  memberId: string
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
  memberId: string
  isThinking: boolean
  /** Optional label like "Processing..." */
  label?: string
}

/** Name tag visibility configuration */
export interface NameTagConfig {
  /** Show name tags for all members */
  showAll: boolean
  /** Show only on hover */
  showOnHover: boolean
  /** Specific member IDs to always show */
  alwaysShowFor: string[]
  /** Specific member IDs to never show */
  neverShowFor: string[]
}

interface OverlayState {
  /** Active speech bubbles */
  speechBubbles: SpeechBubble[]
  /** Members currently in thinking state */
  thinkingStates: Record<string, ThinkingState>
  /** Name tag configuration */
  nameTagConfig: NameTagConfig
  /** Global overlay visibility */
  overlaysVisible: boolean
}

interface OverlayActions {
  /** Show a speech bubble for a member */
  showSpeechBubble: (memberId: string, text: string, duration?: number) => string
  /** Hide a specific speech bubble */
  hideSpeechBubble: (id: string) => void
  /** Hide all speech bubbles for a member */
  hideAllSpeechBubbles: (memberId: string) => void
  /** Set thinking state for a member */
  setThinking: (memberId: string, isThinking: boolean, label?: string) => void
  /** Clear thinking state for a member */
  clearThinking: (memberId: string) => void
  /** Update name tag configuration */
  updateNameTagConfig: (config: Partial<NameTagConfig>) => void
  /** Toggle global overlay visibility */
  setOverlaysVisible: (visible: boolean) => void
  /** Check if name tag should be shown for a member */
  shouldShowNameTag: (memberId: string, isHovered: boolean) => boolean
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

  showSpeechBubble: (memberId, text, duration = 5000) => {
    const id = `bubble-${++bubbleIdCounter}`
    const bubble: SpeechBubble = {
      id,
      memberId,
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

  hideAllSpeechBubbles: (memberId) => {
    set({
      speechBubbles: get().speechBubbles.filter((b) => b.memberId !== memberId),
    })
  },

  setThinking: (memberId, isThinking, label) => {
    set({
      thinkingStates: {
        ...get().thinkingStates,
        [memberId]: { memberId, isThinking, label },
      },
    })
  },

  clearThinking: (memberId) => {
    const { thinkingStates } = get()
    const { [memberId]: _, ...rest } = thinkingStates
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

  shouldShowNameTag: (memberId, isHovered) => {
    const { nameTagConfig, overlaysVisible } = get()

    if (!overlaysVisible) return false
    if (nameTagConfig.neverShowFor.includes(memberId)) return false
    if (nameTagConfig.alwaysShowFor.includes(memberId)) return true
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
 * Selector for speech bubbles for a specific member
 */
export const selectMemberSpeechBubbles = (state: OverlayStore, memberId: string) =>
  state.speechBubbles.filter((b) => b.memberId === memberId)

/**
 * Selector for thinking state of a specific member
 */
export const selectMemberThinking = (state: OverlayStore, memberId: string) =>
  state.thinkingStates[memberId] ?? null
