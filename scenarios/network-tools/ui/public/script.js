import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const APP_ID = 'network-tools'
const API_BASE = '/api/v1'
const DEFAULT_HEADERS = {
  Accept: 'application/json',
  'Content-Type': 'application/json'
}

function initializeIframeBridge() {
  if (typeof window === 'undefined') {
    return
  }

  if (window.parent === window) {
    return
  }

  if (window.__networkToolsBridgeInitialized) {
    return
  }

  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[NetworkTools] Unable to determine parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ appId: APP_ID, parentOrigin })
  window.__networkToolsBridgeInitialized = true
}

initializeIframeBridge()

document.addEventListener('DOMContentLoaded', () => {
  const healthStatus = document.getElementById('health-status')
  const httpForm = document.getElementById('http-form')
  const dnsForm = document.getElementById('dns-form')
  const connectivityForm = document.getElementById('connectivity-form')

  const httpOutput = document.getElementById('http-output')
  const dnsOutput = document.getElementById('dns-output')
  const connectivityOutput = document.getElementById('connectivity-output')

  httpOutput.classList.add('empty')
  httpOutput.textContent = 'Submit an HTTP request to see response details.'
  dnsOutput.classList.add('empty')
  dnsOutput.textContent = 'Run a lookup to view record answers.'
  connectivityOutput.classList.add('empty')
  connectivityOutput.textContent = 'Perform a connectivity check to populate metrics.'

  refreshHealth(healthStatus)
  setInterval(() => refreshHealth(healthStatus), 20_000)

  httpForm.addEventListener('submit', async event => {
    event.preventDefault()
    await handleHttpRequest(httpOutput)
  })

  dnsForm.addEventListener('submit', async event => {
    event.preventDefault()
    await handleDnsLookup(dnsOutput)
  })

  connectivityForm.addEventListener('submit', async event => {
    event.preventDefault()
    await handleConnectivityTest(connectivityOutput)
  })

  addActivity('info', 'UI console ready', 'Requests are proxied through the App Monitor tunnel for safety.')
})

async function refreshHealth(statusCard) {
  try {
    const response = await fetch('/api/health', { headers: DEFAULT_HEADERS })
    const payload = await response.json()

    if (!response.ok) {
      throw new Error(payload.error || 'Health check failed')
    }

    updateStatusCard(statusCard, 'ok', `API healthy • ${payload.service || 'network-tools'}`)
  } catch (error) {
    console.error('[NetworkTools] Health check failed', error)
    updateStatusCard(statusCard, 'error', 'API unreachable through proxy')
    addActivity('error', 'Health check failed', error.message)
  }
}

function updateStatusCard(element, state, message) {
  const indicator = element.querySelector('.status-indicator')
  const text = element.querySelector('.status-text')

  indicator.classList.remove('status-ok', 'status-warning', 'status-error')
  switch (state) {
    case 'ok':
      indicator.classList.add('status-ok')
      break
    case 'error':
      indicator.classList.add('status-error')
      break
    default:
      indicator.classList.add('status-warning')
      break
  }

  text.textContent = message
}

async function handleHttpRequest(output) {
  const urlInput = document.getElementById('http-url')
  const methodInput = document.getElementById('http-method')
  const headersInput = document.getElementById('http-headers')
  const bodyInput = document.getElementById('http-body')
  const followRedirects = document.getElementById('http-follow-redirects')
  const verifySsl = document.getElementById('http-verify-ssl')

  const url = urlInput.value.trim()
  const method = methodInput.value.trim().toUpperCase()

  if (!url) {
    showError(output, 'Provide a valid URL to inspect.')
    return
  }

  let headers
  if (headersInput.value.trim()) {
    headers = safeJsonParse(headersInput.value)
    if (!headers) {
      showError(output, 'Headers must be valid JSON.')
      return
    }
  }

  let bodyContent
  if (bodyInput.value.trim()) {
    bodyContent = tryParseBody(bodyInput.value.trim())
  }

  const payload = {
    url,
    method,
    headers,
    body: bodyContent,
    options: {
      follow_redirects: followRedirects.checked,
      verify_ssl: verifySsl.checked
    }
  }

  setLoading(output, 'Running HTTP request through the tunnel…')

  try {
    const data = await callApi('/network/http', payload)
    renderJson(output, data)
    addActivity('success', `HTTP ${method} ${url}`, 'Request completed successfully.')
  } catch (error) {
    showError(output, error.message)
    addActivity('error', `HTTP ${method} ${url}`, error.message)
  }
}

async function handleDnsLookup(output) {
  const query = document.getElementById('dns-query').value.trim()
  const recordType = document.getElementById('dns-type').value
  const server = document.getElementById('dns-server').value.trim()

  if (!query) {
    showError(output, 'Enter a hostname to resolve.')
    return
  }

  const payload = {
    query,
    record_type: recordType,
    dns_server: server || undefined
  }

  setLoading(output, 'Querying DNS resolvers…')

  try {
    const data = await callApi('/network/dns', payload)
    renderJson(output, data)
    addActivity('success', `DNS ${recordType} ${query}`, 'Lookup completed.')
  } catch (error) {
    showError(output, error.message)
    addActivity('error', `DNS ${recordType} ${query}`, error.message)
  }
}

async function handleConnectivityTest(output) {
  const target = document.getElementById('connectivity-target').value.trim()
  const testType = document.getElementById('connectivity-type').value

  if (!target) {
    showError(output, 'Specify a target host or address.')
    return
  }

  const payload = {
    target,
    test_type: testType,
    options: {
      count: 5,
      timeout_ms: 1500
    }
  }

  setLoading(output, 'Measuring connectivity…')

  try {
    const data = await callApi('/network/test/connectivity', payload)
    renderJson(output, data)
    addActivity('success', `${testType} ${target}`, 'Connectivity metrics captured.')
  } catch (error) {
    showError(output, error.message)
    addActivity('error', `${testType} ${target}`, error.message)
  }
}

async function callApi(path, body) {
  const response = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: DEFAULT_HEADERS,
    body: JSON.stringify(body)
  })

  const payload = await response.json().catch(() => ({}))

  if (!response.ok) {
    const message = payload.error || `Request failed with status ${response.status}`
    throw new Error(message)
  }

  return payload
}

function tryParseBody(raw) {
  const trimmed = raw.trim()
  if (!trimmed) {
    return undefined
  }

  const isJsonLike = (trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))
  if (!isJsonLike) {
    return trimmed
  }

  try {
    return JSON.parse(trimmed)
  } catch (error) {
    console.warn('[NetworkTools] Request body is not valid JSON, sending raw text instead', error)
    return trimmed
  }
}

function safeJsonParse(raw) {
  try {
    return JSON.parse(raw)
  } catch (error) {
    console.warn('[NetworkTools] Failed to parse JSON input', error)
    return null
  }
}

function renderJson(output, payload) {
  output.classList.remove('empty')
  output.textContent = JSON.stringify(payload, null, 2)
}

function setLoading(output, message) {
  output.classList.remove('empty')
  output.textContent = message
}

function showError(output, message) {
  output.classList.remove('empty')
  output.textContent = `⚠️  ${message}`
}

function addActivity(level, summary, details) {
  const list = document.getElementById('activity-log')
  const entry = document.createElement('li')
  const time = document.createElement('time')
  const description = document.createElement('div')
  const detail = document.createElement('div')

  time.dateTime = new Date().toISOString()
  time.textContent = new Date().toLocaleTimeString()
  description.textContent = summary
  detail.textContent = details
  detail.style.color = level === 'error' ? 'var(--danger)' : level === 'success' ? 'var(--success)' : 'var(--text-muted)'

  entry.appendChild(time)
  entry.appendChild(description)
  entry.appendChild(detail)
  list.prepend(entry)

  while (list.children.length > 12) {
    list.removeChild(list.lastChild)
  }
}
