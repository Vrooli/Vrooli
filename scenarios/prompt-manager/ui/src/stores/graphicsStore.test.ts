/**
 * Tests for the graphics settings store.
 *
 * Tests cover:
 * - Tier switching
 * - Individual setting overrides
 * - Config merging
 * - Effective config calculation
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { useGraphicsStore, TIER_CONFIGS } from './graphicsStore'

describe('graphicsStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useGraphicsStore.setState({
      tier: 'medium',
      config: TIER_CONFIGS.medium,
      autoDetect: true,
      overrides: {},
    })
  })

  describe('initial state', () => {
    it('should start with medium tier', () => {
      const state = useGraphicsStore.getState()
      expect(state.tier).toBe('medium')
    })

    it('should have medium tier config', () => {
      const state = useGraphicsStore.getState()
      expect(state.config).toEqual(TIER_CONFIGS.medium)
    })

    it('should have autoDetect enabled', () => {
      const state = useGraphicsStore.getState()
      expect(state.autoDetect).toBe(true)
    })

    it('should have empty overrides', () => {
      const state = useGraphicsStore.getState()
      expect(state.overrides).toEqual({})
    })
  })

  describe('setTier', () => {
    it('should change tier to low', () => {
      useGraphicsStore.getState().setTier('low')

      const state = useGraphicsStore.getState()
      expect(state.tier).toBe('low')
      expect(state.config).toMatchObject(TIER_CONFIGS.low)
    })

    it('should change tier to high', () => {
      useGraphicsStore.getState().setTier('high')

      const state = useGraphicsStore.getState()
      expect(state.tier).toBe('high')
      expect(state.config.shadows).toBe(true)
      expect(state.config.shadowMapSize).toBe(2048)
    })

    it('should change tier to ultra', () => {
      useGraphicsStore.getState().setTier('ultra')

      const state = useGraphicsStore.getState()
      expect(state.tier).toBe('ultra')
      expect(state.config.shadowMapSize).toBe(4096)
    })

    it('should preserve overrides when changing tier', () => {
      useGraphicsStore.getState().setOverride('bloom', false)
      useGraphicsStore.getState().setTier('high')

      const state = useGraphicsStore.getState()
      expect(state.tier).toBe('high')
      expect(state.config.bloom).toBe(false) // Override preserved
    })
  })

  describe('setAutoDetect', () => {
    it('should enable auto-detect', () => {
      useGraphicsStore.getState().setAutoDetect(true)
      expect(useGraphicsStore.getState().autoDetect).toBe(true)
    })

    it('should disable auto-detect', () => {
      useGraphicsStore.getState().setAutoDetect(false)
      expect(useGraphicsStore.getState().autoDetect).toBe(false)
    })
  })

  describe('setOverride', () => {
    it('should override a boolean setting', () => {
      useGraphicsStore.getState().setOverride('shadows', false)

      const state = useGraphicsStore.getState()
      expect(state.overrides.shadows).toBe(false)
      expect(state.config.shadows).toBe(false)
    })

    it('should override a number setting', () => {
      useGraphicsStore.getState().setOverride('shadowMapSize', 512)

      const state = useGraphicsStore.getState()
      expect(state.overrides.shadowMapSize).toBe(512)
      expect(state.config.shadowMapSize).toBe(512)
    })

    it('should override a string setting', () => {
      useGraphicsStore.getState().setOverride('antialiasing', 'fxaa')

      const state = useGraphicsStore.getState()
      expect(state.overrides.antialiasing).toBe('fxaa')
      expect(state.config.antialiasing).toBe('fxaa')
    })

    it('should accumulate multiple overrides', () => {
      useGraphicsStore.getState().setOverride('shadows', false)
      useGraphicsStore.getState().setOverride('bloom', false)
      useGraphicsStore.getState().setOverride('vignette', false)

      const state = useGraphicsStore.getState()
      expect(state.overrides).toEqual({
        shadows: false,
        bloom: false,
        vignette: false,
      })
    })
  })

  describe('clearOverrides', () => {
    it('should clear all overrides', () => {
      useGraphicsStore.getState().setOverride('shadows', false)
      useGraphicsStore.getState().setOverride('bloom', false)

      useGraphicsStore.getState().clearOverrides()

      const state = useGraphicsStore.getState()
      expect(state.overrides).toEqual({})
    })

    it('should restore config to tier defaults', () => {
      useGraphicsStore.getState().setOverride('shadows', false)

      useGraphicsStore.getState().clearOverrides()

      const state = useGraphicsStore.getState()
      expect(state.config).toEqual(TIER_CONFIGS.medium)
    })
  })

  describe('getEffectiveConfig', () => {
    it('should return tier config when no overrides', () => {
      const config = useGraphicsStore.getState().getEffectiveConfig()
      expect(config).toEqual(TIER_CONFIGS.medium)
    })

    it('should merge overrides with tier config', () => {
      useGraphicsStore.getState().setOverride('bloom', false)

      const config = useGraphicsStore.getState().getEffectiveConfig()
      expect(config.bloom).toBe(false)
      expect(config.shadows).toBe(TIER_CONFIGS.medium.shadows)
    })

    it('should use current tier for base config', () => {
      useGraphicsStore.getState().setTier('high')
      useGraphicsStore.getState().setOverride('shadowMapSize', 512)

      const config = useGraphicsStore.getState().getEffectiveConfig()
      expect(config.shadowMapSize).toBe(512)
      expect(config.ssao).toBe(true) // From high tier
    })
  })

  describe('TIER_CONFIGS', () => {
    it('should have valid low tier config', () => {
      expect(TIER_CONFIGS.low).toMatchObject({
        shadows: false,
        postProcessing: false,
        bloom: false,
      })
    })

    it('should have valid medium tier config', () => {
      expect(TIER_CONFIGS.medium).toMatchObject({
        shadows: true,
        postProcessing: true,
        bloom: true,
        ssao: false,
      })
    })

    it('should have valid high tier config', () => {
      expect(TIER_CONFIGS.high).toMatchObject({
        shadows: true,
        shadowMapSize: 2048,
        ssao: true,
        antialiasing: 'smaa',
      })
    })

    it('should have valid ultra tier config', () => {
      expect(TIER_CONFIGS.ultra).toMatchObject({
        shadowMapSize: 4096,
        dpr: 2,
      })
    })
  })
})
