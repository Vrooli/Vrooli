import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // INTEROP-CRITICAL: relative assets remain valid behind proxy/tunnel paths.
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/**/*.d.ts',
        'src/main.tsx',
        'src/test-setup.ts',
        'src/test-utils/**',
        'src/consts/strings.generated.ts',
        'src/i18n/locales/**',
        'src/**/generated/**',
        'src/App.tsx',
        'src/components/BoardController.tsx',
        'src/components/MetricList.tsx',
        'src/components/ThemeProvider.tsx',
        'src/components/ui/**',
        'src/consts/**',
        'src/lib/**',
        'src/main.tsx',
        'src/pages/Broadcast.tsx',
        'src/pages/Forge.tsx',
        'src/pages/Hive.tsx',
        'src/pages/Ledger.tsx',
        'src/pages/Panorama.tsx',
        'src/pages/RoomPage.tsx',
        'src/pages/MissionControl.tsx',
        'src/scenes/missionControl.tsx',
      ],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85
      }
    }
  }
});
