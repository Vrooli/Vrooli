export function initializeIframeBridge(handlers) {
  const CHANNEL = 'web-console'
  const emit = (type, payload) => {
    if (window.parent) {
      window.parent.postMessage({ channel: CHANNEL, type, payload }, '*')
    }
  }

  window.addEventListener('message', async (event) => {
    const message = event.data
    if (!message || message.channel !== CHANNEL) {
      return
    }
    try {
      const handler = handlers[message.type]
      if (handler) {
        await handler(message, emit)
      } else {
        emit('error', { type: 'unknown-command', payload: message })
      }
    } catch (error) {
      emit('error', { type: message.type, message: error instanceof Error ? error.message : 'unknown error' })
    }
  })

  emit('bridge-initialized', { timestamp: Date.now() })
  return { emit }
}
