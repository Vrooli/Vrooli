import { createScenarioServer } from '@vrooli/api-base/server'

const UI_PORT = process.env.UI_PORT
const API_PORT = process.env.API_PORT

const app = createScenarioServer({
  uiPort: UI_PORT,
  apiPort: API_PORT,
  distDir: './dist',
  serviceName: 'agent-inbox',
  corsOrigins: '*',
  verbose: process.env.DEBUG === 'true',
  // Extended timeout for AI completions with web search (can take 60+ seconds)
  proxyTimeoutMs: 180000, // 3 minutes
  embeddedProxy: true,
})

app.listen(UI_PORT, () => {
  console.log(`✅ Agent Inbox UI serving on http://localhost:${UI_PORT}`)
})
