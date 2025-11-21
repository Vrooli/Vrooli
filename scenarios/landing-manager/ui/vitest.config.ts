import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig({ mode: 'test', command: 'serve', ssrBuild: false }),
  defineConfig({
    test: {
      globals: true,
      environment: 'jsdom',
      coverage: {
        provider: 'v8',
        reporter: ['json-summary', 'json', 'text'],
        reportOnFailure: true,
        thresholds: {
          lines: 0,
          functions: 0,
          branches: 0,
          statements: 0
        }
      }
    }
  })
)
