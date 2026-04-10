let startScenarioServer
try {
  ({ startScenarioServer } = await import('@vrooli/api-base/server'))
} catch (error) {
  const isMissingPerfModule =
    error?.code === 'ERR_MODULE_NOT_FOUND' &&
    typeof error?.message === 'string' &&
    error.message.includes('@vrooli/api-base/dist/server/perf.js')

  if (!isMissingPerfModule) {
    throw error
  }

  // Fallback for stale file-dependency installs where server/perf.js is absent.
  ({ startScenarioServer } = await import('../../../packages/api-base/dist/server/template.js'))
}

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'scenario-completeness-scoring',
  corsOrigins: '*',
})
