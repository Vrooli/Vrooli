import { defineProject, mergeConfig } from 'vitest/config';
import baseConfig from './vitest.config.js';

/**
 * Vitest configuration for integration tests
 * These tests typically require longer timeouts due to:
 * - Complex component rendering
 * - Multiple user interactions
 * - Form validation cycles
 * - Async state updates
 */
export default mergeConfig(
    baseConfig,
    defineProject({
        test: {
            // Extend timeouts for integration tests
            testTimeout: 30000, // 30 seconds for complex integration tests
            hookTimeout: 20000, // 20 seconds for setup/teardown
            
            // Include only integration test patterns
            include: [
                'src/**/*.integration.test.{ts,tsx}',
                'src/**/*.roundtrip.test.{ts,tsx}',
                'src/**/*.scenario.test.{ts,tsx}',
                'src/**/*.formHelpers.test.{ts,tsx}', // Form helper tests are integration tests
            ],
            
            // Exclude unit tests
            exclude: [
                ...baseConfig.test?.exclude || [],
                'src/**/*.test.{ts,tsx}',
                '!src/**/*.integration.test.{ts,tsx}',
                '!src/**/*.roundtrip.test.{ts,tsx}',
                '!src/**/*.scenario.test.{ts,tsx}',
                '!src/**/*.formHelpers.test.{ts,tsx}',
            ],
            
            // Use fewer threads for integration tests to avoid resource contention
            poolOptions: {
                threads: {
                    singleThread: false,
                    isolate: true,
                    useAtomics: true,
                    minThreads: 1,
                    maxThreads: 2, // Limit to 2 threads for integration tests
                }
            },
        },
    })
);