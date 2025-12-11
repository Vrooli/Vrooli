/**
 * FPS Controller Tests
 *
 * Unit tests for the adaptive frame rate controller.
 * Tests the pure functions in isolation without any Playwright/WebSocket dependencies.
 */
import {
  processFrame,
  handleTimeout,
  createFpsController,
  getIntervalMs,
  getCurrentFps,
  createInitialState,
  DEFAULT_FPS_CONFIG,
  FpsControllerConfig,
  FpsControllerState,
} from '../../../src/fps';

describe('FPS Controller', () => {
  describe('createInitialState', () => {
    it('should create state with the given initial FPS', () => {
      const state = createInitialState(15);

      expect(state.currentFps).toBe(15);
      expect(state.recentCaptures).toEqual([]);
      expect(state.framesSinceAdjustment).toBe(0);
      expect(state.totalFrames).toBe(0);
      expect(state.lastAdjustment).toBe('none');
    });
  });

  describe('createFpsController', () => {
    it('should create controller with merged config', () => {
      const { state, config } = createFpsController(20, { minFps: 5 });

      expect(state.currentFps).toBe(20);
      expect(config.minFps).toBe(5);
      expect(config.maxFps).toBe(30); // Default, but >= initialFps
      expect(config.targetUtilization).toBe(DEFAULT_FPS_CONFIG.targetUtilization);
    });

    it('should clamp initial FPS to valid range', () => {
      const { state } = createFpsController(100, { maxFps: 30 });
      expect(state.currentFps).toBe(30);

      const { state: state2 } = createFpsController(1, { minFps: 5 });
      expect(state2.currentFps).toBe(5);
    });

    it('should set maxFps to at least initialFps when not specified', () => {
      const { config } = createFpsController(40);
      expect(config.maxFps).toBe(40);
    });
  });

  describe('getIntervalMs', () => {
    it('should calculate correct interval from FPS', () => {
      const state = createInitialState(10);
      expect(getIntervalMs(state)).toBe(100);

      const state2 = createInitialState(30);
      expect(getIntervalMs(state2)).toBe(33);

      const state3 = createInitialState(60);
      expect(getIntervalMs(state3)).toBe(16);
    });
  });

  describe('getCurrentFps', () => {
    it('should return rounded FPS', () => {
      const state: FpsControllerState = {
        ...createInitialState(10),
        currentFps: 15.7,
      };
      expect(getCurrentFps(state)).toBe(16);
    });
  });

  describe('processFrame', () => {
    const config: FpsControllerConfig = {
      ...DEFAULT_FPS_CONFIG,
      minFps: 2,
      maxFps: 30,
      adjustmentInterval: 3,
    };

    it('should not adjust FPS before adjustment interval is reached', () => {
      let state = createInitialState(10);

      // Process 2 frames (less than adjustmentInterval of 3)
      const result1 = processFrame(state, 50, config);
      state = result1.state;
      expect(result1.adjusted).toBe(false);
      expect(result1.newFps).toBe(10);

      const result2 = processFrame(state, 50, config);
      expect(result2.adjusted).toBe(false);
      expect(result2.newFps).toBe(10);
    });

    it('should track capture times in ring buffer', () => {
      let state = createInitialState(10);

      state = processFrame(state, 50, config).state;
      expect(state.recentCaptures).toEqual([50]);

      state = processFrame(state, 60, config).state;
      expect(state.recentCaptures).toEqual([50, 60]);

      state = processFrame(state, 70, config).state;
      expect(state.recentCaptures).toEqual([50, 60, 70]);
    });

    it('should increment frame counters', () => {
      let state = createInitialState(10);

      state = processFrame(state, 50, config).state;
      expect(state.totalFrames).toBe(1);
      expect(state.framesSinceAdjustment).toBe(1);

      state = processFrame(state, 50, config).state;
      expect(state.totalFrames).toBe(2);
      expect(state.framesSinceAdjustment).toBe(2);
    });

    describe('FPS increase (fast captures)', () => {
      it('should increase FPS when captures are fast', () => {
        let state = createInitialState(10);
        // At 10 FPS, interval = 100ms, target capture = 70ms (at 0.7 utilization)
        // Increase threshold: 70 * 0.6 = 42ms
        // If captures are < 42ms, should increase

        // Process 3 fast frames (< 42ms)
        state = processFrame(state, 30, config).state;
        state = processFrame(state, 30, config).state;
        const result = processFrame(state, 30, config);

        expect(result.adjusted).toBe(true);
        expect(result.newFps).toBeGreaterThan(10);
        expect(result.state.lastAdjustment).toBe('up');
        expect(result.diagnostics?.reason).toBe('too_fast');
      });

      it('should not increase FPS above maxFps', () => {
        const lowMaxConfig = { ...config, maxFps: 10 };
        let state = createInitialState(10);

        // Process 3 very fast frames
        state = processFrame(state, 10, lowMaxConfig).state;
        state = processFrame(state, 10, lowMaxConfig).state;
        const result = processFrame(state, 10, lowMaxConfig);

        // Should not increase since we're at max
        expect(result.newFps).toBe(10);
      });
    });

    describe('FPS decrease (slow captures)', () => {
      it('should decrease FPS when captures are slow', () => {
        let state = createInitialState(10);
        // At 10 FPS, interval = 100ms, target capture = 70ms
        // Decrease threshold: 70 * 1.15 = 80.5ms
        // If captures are > 80.5ms, should decrease

        // Process 3 slow frames (> 80ms)
        state = processFrame(state, 90, config).state;
        state = processFrame(state, 90, config).state;
        const result = processFrame(state, 90, config);

        expect(result.adjusted).toBe(true);
        expect(result.newFps).toBeLessThan(10);
        expect(result.state.lastAdjustment).toBe('down');
        expect(result.diagnostics?.reason).toBe('too_slow');
      });

      it('should not decrease FPS below minFps', () => {
        let state = createInitialState(3);
        const lowMinConfig = { ...config, minFps: 2 };

        // Process 3 very slow frames
        state = processFrame(state, 500, lowMinConfig).state;
        state = processFrame(state, 500, lowMinConfig).state;
        const result = processFrame(state, 500, lowMinConfig);

        expect(result.newFps).toBeGreaterThanOrEqual(2);
      });
    });

    describe('stable FPS (captures in sweet spot)', () => {
      it('should not adjust FPS when captures are in acceptable range', () => {
        let state = createInitialState(10);
        // At 10 FPS, interval = 100ms, target capture = 70ms
        // Sweet spot: 42ms < capture < 80.5ms (roughly)

        // Process 3 frames in sweet spot (~60ms)
        state = processFrame(state, 60, config).state;
        state = processFrame(state, 60, config).state;
        const result = processFrame(state, 60, config);

        expect(result.adjusted).toBe(false);
        expect(result.newFps).toBe(10);
      });
    });

    describe('smoothing behavior', () => {
      it('should apply smoothing to FPS changes', () => {
        const smoothConfig = { ...config, smoothing: 0.5 };
        let state = createInitialState(10);

        // Process 3 very fast frames that would ideally suggest much higher FPS
        state = processFrame(state, 10, smoothConfig).state;
        state = processFrame(state, 10, smoothConfig).state;
        const result = processFrame(state, 10, smoothConfig);

        // With smoothing, we shouldn't jump all the way to ideal FPS
        // Should increase but not dramatically
        expect(result.adjusted).toBe(true);
        expect(result.newFps).toBeGreaterThan(10);
        expect(result.newFps).toBeLessThan(30); // Not max immediately
      });
    });

    describe('diagnostics', () => {
      it('should provide diagnostics when adjustment occurs', () => {
        let state = createInitialState(10);

        state = processFrame(state, 90, config).state;
        state = processFrame(state, 90, config).state;
        const result = processFrame(state, 90, config);

        expect(result.adjusted).toBe(true);
        expect(result.diagnostics).toBeDefined();
        expect(result.diagnostics?.previousFps).toBe(10);
        expect(result.diagnostics?.avgCaptureMs).toBe(90);
        expect(result.diagnostics?.targetCaptureMs).toBeGreaterThan(0);
        expect(result.diagnostics?.idealFps).toBeGreaterThan(0);
      });

      it('should not provide diagnostics when no adjustment', () => {
        let state = createInitialState(10);

        const result = processFrame(state, 60, config);

        expect(result.adjusted).toBe(false);
        expect(result.diagnostics).toBeUndefined();
      });
    });

    describe('ring buffer limits', () => {
      it('should limit ring buffer to 10 entries', () => {
        let state = createInitialState(10);

        // Process 15 frames
        for (let i = 0; i < 15; i++) {
          state = processFrame(state, 50 + i, config).state;
        }

        expect(state.recentCaptures.length).toBe(10);
        // Should have dropped earliest entries
        expect(state.recentCaptures[0]).toBe(55); // 50+5, since first 5 were dropped
      });
    });
  });

  describe('handleTimeout', () => {
    const config: FpsControllerConfig = {
      ...DEFAULT_FPS_CONFIG,
      minFps: 2,
      maxFps: 30,
    };

    it('should immediately reduce FPS on timeout', () => {
      const state = createInitialState(20);

      const result = handleTimeout(state, 200, config);

      expect(result.adjusted).toBe(true);
      expect(result.newFps).toBeLessThan(20);
      expect(result.state.lastAdjustment).toBe('down');
    });

    it('should reduce by at least 2 FPS or 25%', () => {
      // At 20 FPS, 25% = 5, which is > 2, so should reduce by 5
      const state = createInitialState(20);
      const result = handleTimeout(state, 200, config);
      expect(result.newFps).toBeLessThanOrEqual(15);

      // At 6 FPS, 25% = 1.5, which is < 2, so should reduce by 2
      const state2 = createInitialState(6);
      const result2 = handleTimeout(state2, 200, config);
      expect(result2.newFps).toBeLessThanOrEqual(4);
    });

    it('should not reduce FPS below minFps', () => {
      const state = createInitialState(3);

      const result = handleTimeout(state, 200, config);

      expect(result.newFps).toBeGreaterThanOrEqual(2);
    });

    it('should reset framesSinceAdjustment', () => {
      const state: FpsControllerState = {
        ...createInitialState(20),
        framesSinceAdjustment: 5,
      };

      const result = handleTimeout(state, 200, config);

      expect(result.state.framesSinceAdjustment).toBe(0);
    });

    it('should add timeout to capture buffer', () => {
      const state = createInitialState(20);

      const result = handleTimeout(state, 200, config);

      expect(result.state.recentCaptures).toContain(200);
    });

    it('should provide diagnostics', () => {
      const state = createInitialState(20);

      const result = handleTimeout(state, 200, config);

      expect(result.diagnostics).toBeDefined();
      expect(result.diagnostics?.previousFps).toBe(20);
      expect(result.diagnostics?.avgCaptureMs).toBe(200);
      expect(result.diagnostics?.reason).toBe('too_slow');
    });
  });

  describe('real-world scenarios', () => {
    const config: FpsControllerConfig = {
      ...DEFAULT_FPS_CONFIG,
      minFps: 2,
      maxFps: 30,
      adjustmentInterval: 3,
    };

    it('should stabilize around achievable FPS for consistent capture times', () => {
      let state = createInitialState(15);

      // Simulate 30 frames with consistent 80ms capture time
      // At ~12 FPS (83ms interval), 80ms capture would use ~96% of budget
      // Should settle somewhere around 10-12 FPS
      for (let i = 0; i < 30; i++) {
        const result = processFrame(state, 80, config);
        state = result.state;
      }

      // Should have stabilized to a sustainable rate
      const finalFps = getCurrentFps(state);
      const interval = getIntervalMs(state);

      // At this FPS, capture should use less than 100% of interval
      expect(80 / interval).toBeLessThan(1);
      expect(finalFps).toBeGreaterThanOrEqual(config.minFps);
    });

    it('should ramp up when page becomes faster', () => {
      let state = createInitialState(10);

      // First, settle at a low FPS with slow captures
      for (let i = 0; i < 15; i++) {
        state = processFrame(state, 100, config).state;
      }
      const lowFps = getCurrentFps(state);

      // Now simulate faster captures
      for (let i = 0; i < 15; i++) {
        state = processFrame(state, 30, config).state;
      }
      const highFps = getCurrentFps(state);

      expect(highFps).toBeGreaterThan(lowFps);
    });

    it('should handle variable capture times gracefully', () => {
      let state = createInitialState(15);

      // Simulate variable capture times (40-120ms range)
      const captureTimes = [40, 80, 60, 120, 50, 90, 70, 100, 55, 85];

      for (const captureTime of captureTimes) {
        state = processFrame(state, captureTime, config).state;
      }

      // Should remain within bounds and be reasonable
      const fps = getCurrentFps(state);
      expect(fps).toBeGreaterThanOrEqual(config.minFps);
      expect(fps).toBeLessThanOrEqual(config.maxFps);
    });

    it('should recover after timeouts', () => {
      let state = createInitialState(15);

      // Get into steady state
      for (let i = 0; i < 10; i++) {
        state = processFrame(state, 50, config).state;
      }
      const preTimeoutFps = getCurrentFps(state);

      // Simulate timeout
      state = handleTimeout(state, 200, config).state;
      const postTimeoutFps = getCurrentFps(state);

      expect(postTimeoutFps).toBeLessThan(preTimeoutFps);

      // Recover with fast captures
      for (let i = 0; i < 15; i++) {
        state = processFrame(state, 30, config).state;
      }
      const recoveredFps = getCurrentFps(state);

      expect(recoveredFps).toBeGreaterThan(postTimeoutFps);
    });
  });

  describe('edge cases', () => {
    it('should handle zero capture time', () => {
      const state = createInitialState(10);
      const result = processFrame(state, 0, DEFAULT_FPS_CONFIG);

      expect(result.state.recentCaptures).toContain(0);
      expect(result.newFps).toBeGreaterThanOrEqual(DEFAULT_FPS_CONFIG.minFps);
    });

    it('should handle very large capture times', () => {
      const state = createInitialState(10);
      const result = processFrame(state, 10000, DEFAULT_FPS_CONFIG);

      expect(result.state.recentCaptures).toContain(10000);
    });

    it('should handle fractional FPS correctly', () => {
      // State can have fractional FPS internally
      const state: FpsControllerState = {
        ...createInitialState(10),
        currentFps: 10.5,
      };

      const result = processFrame(state, 50, DEFAULT_FPS_CONFIG);

      // intervalMs should be calculated from fractional FPS
      expect(result.intervalMs).toBe(Math.floor(1000 / 10.5));
    });

    it('should not mutate input state', () => {
      const state = createInitialState(10);
      const originalState = { ...state, recentCaptures: [...state.recentCaptures] };

      processFrame(state, 50, DEFAULT_FPS_CONFIG);

      expect(state).toEqual(originalState);
    });
  });
});
