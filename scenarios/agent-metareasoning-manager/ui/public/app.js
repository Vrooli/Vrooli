import { initIframeBridgeChild } from '/bridge/iframeBridgeChild.js'

const BRIDGE_FLAG = '__metareasoningBridgeInitialized'
const STATE = {
  apiBase: '/api',
  workflows: [],
  filteredWorkflows: [],
  lastLatencyMs: null,
  activePanel: 'overview'
}

function safeRead(fn, fallback) {
  try {
    const value = fn()
    return typeof value === 'undefined' ? fallback : value
  } catch (error) {
    console.warn('[agent-metareasoning-manager][ui] Proxy metadata read failed', error)
    return fallback
  }
}

function deriveParentOrigin() {
  const referrer = document.referrer
  if (!referrer) {
    return undefined
  }

  try {
    return new URL(referrer).origin
  } catch (error) {
    console.warn('[agent-metareasoning-manager][ui] Unable to parse parent origin', error)
    return undefined
  }
}

function bootstrapIframeBridge() {
  if (typeof window === 'undefined') {
    return
  }
  if (window.parent === window) {
    return
  }
  if (window[BRIDGE_FLAG]) {
    return
  }

  const parentOrigin = deriveParentOrigin()
  initIframeBridgeChild({ parentOrigin, appId: 'agent-metareasoning-manager' })
  window[BRIDGE_FLAG] = true
}

function readProxyPath(candidate) {
  if (!candidate || typeof candidate !== 'object') {
    return undefined
  }

  const possibleKeys = ['path', 'proxyPath', 'url', 'origin', 'base', 'baseUrl']
  for (const key of possibleKeys) {
    const value = candidate[key]
    if (typeof value === 'string' && value.trim().length > 0) {
      return value.trim()
    }
  }
  return undefined
}

function deriveApiBase() {
  if (typeof window === 'undefined') {
    return '/api'
  }

  const proxyInfo = safeRead(() => window.__APP_MONITOR_PROXY_INFO__)
  if (proxyInfo) {
    const primary = safeRead(() => proxyInfo.primary)
    const proxyPath = readProxyPath(primary)
    if (proxyPath) {
      return proxyPath.startsWith('http') ? proxyPath : normalizeRelativePath(proxyPath)
    }

    const routes = safeRead(() => proxyInfo.routes)
    if (Array.isArray(routes)) {
      for (const route of routes) {
        const found = readProxyPath(route)
        if (found) {
          return found.startsWith('http') ? found : normalizeRelativePath(found)
        }
      }
    }
  }

  const fallbackIndex = safeRead(() => window.__APP_MONITOR_PROXY_INDEX__)
  if (fallbackIndex && typeof fallbackIndex === 'object') {
    const first = Array.isArray(fallbackIndex) ? fallbackIndex[0] : fallbackIndex.primary
    const proxyPath = readProxyPath(first)
    if (proxyPath) {
      return proxyPath.startsWith('http') ? proxyPath : normalizeRelativePath(proxyPath)
    }
  }

  return '/api'
}

function normalizeRelativePath(pathCandidate) {
  if (typeof pathCandidate !== 'string' || pathCandidate.length === 0) {
    return '/api'
  }
  if (pathCandidate.startsWith('http://') || pathCandidate.startsWith('https://')) {
    return pathCandidate
  }
  const trimmed = pathCandidate.startsWith('/') ? pathCandidate : `/${pathCandidate}`
  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed
}

function joinUrl(base, path) {
  const safeBase = typeof base === 'string' && base.length > 0 ? base : '/api'
  const safePath = typeof path === 'string' && path.length > 0 ? path : '/'

  const normalizedBase = safeBase.endsWith('/') ? safeBase.slice(0, -1) : safeBase
  const normalizedPath = safePath.startsWith('/') ? safePath : `/${safePath}`

  if (normalizedBase === '.') {
    return normalizedPath
  }

  return `${normalizedBase}${normalizedPath}`
}

async function fetchJson(path, options = {}) {
  const url = joinUrl(STATE.apiBase, path)
  const response = await fetch(url, {
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      ...options.headers
    },
    ...options
  })

  if (!response.ok) {
    const text = await response.text().catch(() => '')
    const error = new Error(`Request failed with status ${response.status}`)
    error.status = response.status
    error.body = text
    throw error
  }

  if (response.status === 204) {
    return null
  }

  return response.json()
}

function updateHealthUI({ status, version, latencyMs, catalogSize }) {
  const chip = document.getElementById('health-chip')
  const statusEl = document.getElementById('health-status')
  const versionEl = document.getElementById('health-version')
  const latencyEl = document.getElementById('health-latency')
  const catalogEl = document.getElementById('health-catalog')

  if (chip) {
    const healthy = status && status.toLowerCase() === 'healthy'
    chip.textContent = healthy ? 'Healthy · API responding' : `Status: ${status}`
    chip.dataset.state = healthy ? 'healthy' : 'degraded'
  }

  if (statusEl) {
    statusEl.textContent = status ? status.toUpperCase() : '—'
  }

  if (versionEl) {
    versionEl.textContent = version || '—'
  }

  if (latencyEl) {
    latencyEl.textContent = typeof latencyMs === 'number' ? `${Math.round(latencyMs)} ms` : '—'
  }

  if (catalogEl) {
    catalogEl.textContent = typeof catalogSize === 'number' ? String(catalogSize) : '—'
  }
}

function renderWorkflows() {
  const grid = document.getElementById('workflow-grid')
  const empty = document.getElementById('workflow-empty')
  if (!grid) {
    return
  }

  grid.innerHTML = ''

  const items = STATE.filteredWorkflows.length ? STATE.filteredWorkflows : STATE.workflows

  if (!items || items.length === 0) {
    if (empty) {
      empty.textContent = 'No workflows available. Connect Windmill or n8n resources to populate the catalog.'
      empty.classList.remove('hidden')
    }
    return
  }

  if (empty) {
    empty.textContent = ''
    empty.classList.add('hidden')
  }

  const fragment = document.createDocumentFragment()
  items.forEach(workflow => {
    const card = document.createElement('article')
    card.className = 'workflow-card'

    const title = document.createElement('h4')
    title.textContent = workflow.name || 'Untitled Workflow'
    card.appendChild(title)

    const summary = document.createElement('p')
    summary.textContent = workflow.description || 'No description provided.'
    summary.className = 'workflow-summary'
    card.appendChild(summary)

    const meta = document.createElement('div')
    meta.className = 'workflow-meta'
    meta.innerHTML = `<span>${(workflow.platform || 'unknown').toUpperCase()}</span><span>${workflow.usage_count || 0} runs</span>`
    card.appendChild(meta)

    if (Array.isArray(workflow.tags) && workflow.tags.length > 0) {
      const tags = document.createElement('div')
      tags.className = 'workflow-tags'
      workflow.tags.forEach(tag => {
        const chip = document.createElement('span')
        chip.className = 'workflow-tag'
        chip.textContent = tag
        tags.appendChild(chip)
      })
      card.appendChild(tags)
    }

    const actions = document.createElement('div')
    actions.className = 'workflow-actions'

    const detailsBtn = document.createElement('button')
    detailsBtn.className = 'secondary-button'
    detailsBtn.type = 'button'
    detailsBtn.textContent = 'View Payload'
    detailsBtn.addEventListener('click', () => {
      const payload = {
        platform: workflow.platform,
        platformId: workflow.platform_id,
        description: workflow.description,
        tags: workflow.tags,
        capabilities: workflow.capabilities
      }
      const pretty = JSON.stringify(payload, null, 2)
      const output = document.getElementById('analysis-output')
      if (output) {
        output.innerHTML = ''
        const heading = document.createElement('h4')
        heading.className = 'analysis-heading'
        heading.textContent = `Workflow: ${workflow.name}`
        const pre = document.createElement('pre')
        pre.textContent = pretty
        output.appendChild(heading)
        output.appendChild(pre)
        switchPanel('insights')
      }
    })
    actions.appendChild(detailsBtn)

    const runBtn = document.createElement('button')
    runBtn.className = 'primary-button'
    runBtn.type = 'button'
    runBtn.textContent = 'Execute'
    runBtn.addEventListener('click', async () => {
      runBtn.disabled = true
      runBtn.textContent = 'Executing…'
      try {
        const endpoint = `/execute/${workflow.platform}/${workflow.platform_id}`
        const result = await fetchJson(endpoint, { method: 'POST', body: JSON.stringify({}) })
        displayAnalysisResult({
          heading: `Execution: ${workflow.name}`,
          payload: result
        })
        switchPanel('insights')
      } catch (error) {
        displayError(`Failed to execute workflow: ${error.body || error.message}`)
      } finally {
        runBtn.disabled = false
        runBtn.textContent = 'Execute'
      }
    })
    actions.appendChild(runBtn)

    card.appendChild(actions)
    fragment.appendChild(card)
  })

  grid.appendChild(fragment)
}

function displayAnalysisResult({ heading, payload }) {
  const container = document.getElementById('analysis-output')
  if (!container) {
    return
  }

  container.innerHTML = ''
  const title = document.createElement('h4')
  title.className = 'analysis-heading'
  title.textContent = heading
  const pre = document.createElement('pre')
  pre.textContent = JSON.stringify(payload, null, 2)
  container.appendChild(title)
  container.appendChild(pre)
}

function displayError(message) {
  const container = document.getElementById('analysis-output')
  if (!container) {
    return
  }

  container.innerHTML = ''
  const title = document.createElement('h4')
  title.className = 'analysis-heading'
  title.textContent = 'Something went wrong'
  const notice = document.createElement('div')
  notice.className = 'analysis-placeholder'
  notice.textContent = message
  container.appendChild(title)
  container.appendChild(notice)
}

function attachNavigationHandlers() {
  const buttons = document.querySelectorAll('.nav-item')
  buttons.forEach(button => {
    button.addEventListener('click', () => {
      const target = button.getAttribute('data-target')
      if (!target) {
        return
      }
      switchPanel(target)
    })
  })
}

function switchPanel(panelId) {
  const panels = document.querySelectorAll('.section')
  panels.forEach(panel => {
    const isTarget = panel.getAttribute('data-panel') === panelId
    panel.classList.toggle('hidden', !isTarget)
  })

  const buttons = document.querySelectorAll('.nav-item')
  buttons.forEach(button => {
    const isTarget = button.getAttribute('data-target') === panelId
    button.setAttribute('aria-pressed', isTarget ? 'true' : 'false')
  })

  STATE.activePanel = panelId
}

function attachFilters() {
  const search = document.getElementById('workflow-search')
  const platformFilter = document.getElementById('platform-filter')

  const filter = () => {
    const term = search ? search.value.trim().toLowerCase() : ''
    const platform = platformFilter ? platformFilter.value : 'all'

    STATE.filteredWorkflows = STATE.workflows.filter(item => {
      const matchesPlatform = platform === 'all' || (item.platform || '').toLowerCase() === platform
      if (!term) {
        return matchesPlatform
      }
      const haystack = [item.name, item.description, (item.tags || []).join(' '), item.platform].join(' ').toLowerCase()
      return matchesPlatform && haystack.includes(term)
    })

    renderWorkflows()
  }

  if (search) {
    search.addEventListener('input', filter)
  }

  if (platformFilter) {
    platformFilter.addEventListener('change', filter)
  }
}

function attachAnalysisHandler() {
  const form = document.getElementById('analysis-form')
  if (!form) {
    return
  }

  form.addEventListener('submit', async event => {
    event.preventDefault()

    const pattern = document.getElementById('analysis-pattern')?.value || 'pros-cons'
    const topic = document.getElementById('analysis-topic')?.value?.trim()
    const context = document.getElementById('analysis-context')?.value?.trim()

    if (!topic) {
      displayError('Please describe the decision or question you want to analyze.')
      return
    }

    const payload = {
      input: topic,
      context,
      constraints: context,
      metadata: {
        interface: 'ui',
        source: 'agent-metareasoning-manager'
      }
    }

    let endpoint = '/reasoning'

    switch (pattern) {
      case 'pros-cons':
        endpoint = '/reasoning/pros-cons'
        break
      case 'swot':
        endpoint = '/reasoning/swot'
        break
      case 'risk-assessment':
        endpoint = '/reasoning/risk-assessment'
        break
      case 'self-review':
        endpoint = '/reasoning/self-review'
        break
      case 'chain':
        endpoint = '/reasoning/chain'
        payload.chain_type = 'metacognitive'
        break
      default:
        endpoint = '/reasoning'
    }

    const submitButton = form.querySelector('button[type="submit"]')
    if (submitButton) {
      submitButton.disabled = true
      submitButton.textContent = 'Running analysis…'
    }

    try {
      const result = await fetchJson(endpoint, { method: 'POST', body: JSON.stringify(payload) })
      displayAnalysisResult({
        heading: `Analysis: ${pattern.replace('-', ' ').toUpperCase()}`,
        payload: result
      })
      switchPanel('insights')
    } catch (error) {
      displayError(`Analysis failed: ${error.body || error.message}`)
    } finally {
      if (submitButton) {
        submitButton.disabled = false
        submitButton.textContent = 'Run Structured Analysis'
      }
    }
  })
}

async function loadHealthAndCatalog() {
  try {
    const healthStart = performance.now()
    const health = await fetchJson('/health')
    const latency = performance.now() - healthStart
    STATE.lastLatencyMs = latency

    updateHealthUI({
      status: health?.status,
      version: health?.version,
      latencyMs: latency,
      catalogSize: STATE.workflows?.length || 0
    })
  } catch (error) {
    displayError(`Unable to reach API health endpoint: ${error.body || error.message}`)
    updateHealthUI({
      status: 'offline',
      version: 'unknown',
      latencyMs: null,
      catalogSize: STATE.workflows?.length || 0
    })
  }
}

async function loadWorkflows() {
  try {
    const start = performance.now()
    const workflows = await fetchJson('/workflows')
    const latency = performance.now() - start
    STATE.workflows = Array.isArray(workflows) ? workflows : []
    STATE.filteredWorkflows = []
    STATE.lastLatencyMs = latency

    renderWorkflows()
    updateHealthUI({
      status: 'healthy',
      version: null,
      latencyMs: latency,
      catalogSize: STATE.workflows.length
    })
  } catch (error) {
    console.error('[agent-metareasoning-manager][ui] Failed to load workflows', error)
    const empty = document.getElementById('workflow-empty')
    if (empty) {
      empty.textContent = 'Unable to load workflow catalog. Ensure the API is running and resources are initialized.'
      empty.classList.remove('hidden')
    }
  }
}

function initialize() {
  bootstrapIframeBridge()
  STATE.apiBase = deriveApiBase()
  attachNavigationHandlers()
  attachFilters()
  attachAnalysisHandler()
  Promise.all([loadWorkflows(), loadHealthAndCatalog()])
}

document.addEventListener('DOMContentLoaded', initialize)
