/**
 * Mobile keyboard toolbar - provides quick keys and AI command generation
 * Inspired by Terminus app UX for mobile terminal access
 */

/**
 * Mobile toolbar state
 */
const toolbarState = {
  visible: false,
  mode: 'floating', // 'disabled', 'floating', or 'top'
  modifiers: {
    ctrl: false,
    alt: false
  },
  contextMode: false,
  selectedLines: [],
  aiInputValue: ''
}

/**
 * Check if device is mobile/touch
 */
function isMobileDevice() {
  return (
    'ontouchstart' in window ||
    navigator.maxTouchPoints > 0 ||
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
  )
}

/**
 * Create toolbar DOM structure
 */
function createToolbarDOM() {
  const toolbar = document.createElement('div')
  toolbar.id = 'mobileKeyboardToolbar'
  toolbar.className = 'mobile-toolbar hidden'
  toolbar.innerHTML = `
    <div class="mobile-toolbar-content">
      <!-- AI Command Generation Row -->
      <div class="mobile-toolbar-ai-row">
        <input
          type="text"
          id="aiCommandInput"
          class="ai-command-input"
          placeholder="Ask AI to generate a command"
          autocomplete="off"
        />
        <button type="button" id="aiGenerateBtn" class="ai-generate-btn" title="Generate command">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/>
          </svg>
        </button>
        <button type="button" id="toolbarMenuToggle" class="toolbar-menu-toggle hidden" title="Open menu" aria-label="Open session details">
          <span class="drawer-icon-lines">
            <span></span>
            <span></span>
            <span></span>
          </span>
        </button>
      </div>

      <!-- Context Mode Row (shown when AI input has text) -->
      <div class="mobile-toolbar-context-row hidden">
        <button type="button" id="contextModeBtn" class="toolbar-btn context-btn">
          <span class="context-icon">⚡</span>
          Context (0)
        </button>
      </div>

      <!-- Quick Keys Row (default) -->
      <div class="mobile-toolbar-keys-row">
        <button type="button" class="toolbar-btn quick-key" data-key="Escape" title="Escape">esc</button>
        <button type="button" class="toolbar-btn quick-key" data-key="Tab" title="Tab">tab</button>
        <button type="button" class="toolbar-btn modifier-key" data-modifier="ctrl" title="Control">ctrl</button>
        <button type="button" class="toolbar-btn modifier-key" data-modifier="alt" title="Alt">alt</button>
        <button type="button" class="toolbar-btn quick-key" data-key="/" title="Slash">/</button>
        <button type="button" class="toolbar-btn quick-key" data-key="|" title="Pipe">|</button>
        <button type="button" class="toolbar-btn quick-key" data-key="~" title="Tilde">~</button>
        <button type="button" class="toolbar-btn quick-key" data-key="-" title="Dash">-</button>
        <button type="button" class="toolbar-btn quick-key" data-key="\\x03" data-display="^C" title="Ctrl+C">^C</button>
      </div>
    </div>
  `
  document.body.appendChild(toolbar)
  return toolbar
}

/**
 * Initialize mobile toolbar
 */
export function initializeMobileToolbar(getActiveTabFn, sendKeyToTerminalFn) {
  const isMobile = isMobileDevice()

  // Create toolbar for both mobile and desktop
  const toolbar = createToolbarDOM()

  // Add desktop-specific class if not mobile
  if (!isMobile) {
    toolbar.classList.add('desktop-mode')
  }

  const aiInput = document.getElementById('aiCommandInput')
  const aiGenerateBtn = document.getElementById('aiGenerateBtn')
  const contextRow = toolbar.querySelector('.mobile-toolbar-context-row')
  const keysRow = toolbar.querySelector('.mobile-toolbar-keys-row')
  const contextModeBtn = document.getElementById('contextModeBtn')
  const toolbarMenuToggle = document.getElementById('toolbarMenuToggle')

  // Store toolbar and keyboard detection cleanup function
  let keyboardDetectionCleanup = null

  // Handle AI input changes
  aiInput?.addEventListener('input', (e) => {
    toolbarState.aiInputValue = e.target.value
    updateToolbarUI(contextRow, keysRow, contextModeBtn)
  })

  // Handle AI generate button
  aiGenerateBtn?.addEventListener('click', () => {
    handleAIGenerate(aiInput, getActiveTabFn, sendKeyToTerminalFn)
  })

  // Handle context mode toggle
  contextModeBtn?.addEventListener('click', () => {
    toolbarState.contextMode = !toolbarState.contextMode
    updateToolbarUI(contextRow, keysRow, contextModeBtn)
    toggleContextSelectionMode(getActiveTabFn, contextRow, keysRow, contextModeBtn)
  })

  // Handle toolbar menu toggle (opens main drawer)
  toolbarMenuToggle?.addEventListener('click', () => {
    const drawerToggle = document.getElementById('drawerToggle')
    if (drawerToggle) {
      drawerToggle.click()
    }
  })

  // Handle quick keys
  toolbar.querySelectorAll('.quick-key').forEach(btn => {
    btn.addEventListener('click', () => {
      const key = btn.dataset.key
      sendQuickKey(key, sendKeyToTerminalFn)
    })
  })

  // Handle modifier keys (toggle)
  toolbar.querySelectorAll('.modifier-key').forEach(btn => {
    btn.addEventListener('click', () => {
      const modifier = btn.dataset.modifier
      toggleModifier(modifier, btn)
    })

    // Long press to show hints
    let longPressTimer = null
    btn.addEventListener('pointerdown', (e) => {
      longPressTimer = setTimeout(() => {
        showModifierHints(btn.dataset.modifier, btn)
      }, 800)
    })

    btn.addEventListener('pointerup', () => {
      if (longPressTimer) clearTimeout(longPressTimer)
    })

    btn.addEventListener('pointerleave', () => {
      if (longPressTimer) clearTimeout(longPressTimer)
    })
  })

  // Listen for regular keypresses to apply modifiers
  document.addEventListener('keydown', (e) => {
    if (toolbarState.modifiers.ctrl || toolbarState.modifiers.alt) {
      handleModifiedKey(e, sendKeyToTerminalFn)
    }
  })

  // Apply initial mode from window global (set during workspace load)
  const initialMode = window.__keyboardToolbarMode || 'floating'

  // Setup mode with keyboard detection
  const setupMode = (mode) => {
    // Clean up previous keyboard detection
    if (keyboardDetectionCleanup) {
      keyboardDetectionCleanup()
      keyboardDetectionCleanup = null
    }

    // Update mode
    toolbarState.mode = mode

    // Remove all mode-related classes
    toolbar.classList.remove('mode-disabled', 'mode-floating', 'mode-top')

    // Desktop mode: Hide toolbar by default
    if (!isMobile) {
      // On desktop, toolbar should be disabled by default
      toolbar.classList.add('mode-disabled', 'hidden')
      console.log(`Toolbar mode set to: disabled (desktop - use mobile view to test)`)
      return
    }

    // Mobile mode: Full mode support
    switch (mode) {
      case 'disabled':
        toolbar.classList.add('mode-disabled', 'hidden')
        break
      case 'floating':
        toolbar.classList.add('mode-floating')
        // Setup keyboard detection for floating mode
        keyboardDetectionCleanup = setupKeyboardDetection(toolbar)
        break
      case 'top':
        toolbar.classList.add('mode-top')
        // For top mode, position it at the top replacing the header
        toolbar.style.position = 'fixed'
        toolbar.style.top = '0'
        toolbar.style.bottom = 'auto'
        toolbar.style.left = '0'
        toolbar.style.right = '0'
        showToolbar(toolbar)
        break
    }

    console.log(`Toolbar mode set to: ${mode}`)
  }

  setupMode(initialMode)

  const deviceType = isMobile ? 'mobile' : 'desktop'
  console.log(`Mobile toolbar initialized with mode: ${initialMode} (${deviceType})`)

  return {
    show: () => showToolbar(toolbar),
    hide: () => hideToolbar(toolbar),
    getContextLines: () => toolbarState.selectedLines,
    setMode: (mode) => setupMode(mode)
  }
}

/**
 * Setup keyboard visibility detection
 * iOS 26 has major bugs with visualViewport.offsetTop and fixed positioning
 * This uses a more robust approach with position: absolute
 *
 * Returns a cleanup function to remove event listeners
 */
function setupKeyboardDetection(toolbar) {

  // Detect iOS
  const isIOS = /iPhone|iPad|iPod/.test(navigator.userAgent) ||
                (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)

  if (window.visualViewport && isIOS) {
    // iOS-specific handling due to iOS 26 bugs
    // Switch toolbar to absolute positioning on iOS
    toolbar.style.position = 'absolute'

    const updateToolbarPosition = () => {
      const viewport = window.visualViewport
      const keyboardHeight = window.innerHeight - viewport.height
      const keyboardVisible = keyboardHeight > 100

      // Debug logging
      console.log('[iOS Toolbar] Update:', {
        windowHeight: window.innerHeight,
        viewportHeight: viewport.height,
        viewportOffsetTop: viewport.offsetTop,
        keyboardHeight,
        keyboardVisible,
        scrollY: window.pageYOffset || document.documentElement.scrollTop
      })

      if (keyboardVisible) {
        // Calculate position accounting for page scroll and viewport
        const scrollY = window.pageYOffset || document.documentElement.scrollTop
        const toolbarHeight = toolbar.offsetHeight || 150
        const toolbarTop = scrollY + viewport.height - toolbarHeight

        console.log('[iOS Toolbar] Positioning above keyboard:', {
          toolbarTop,
          toolbarHeight,
          calculatedPosition: Math.max(0, toolbarTop)
        })

        toolbar.style.top = `${Math.max(0, toolbarTop)}px`
        toolbar.style.bottom = 'auto'
        toolbar.style.position = 'absolute'
        showToolbar(toolbar)
      } else {
        // Reset to fixed bottom when keyboard is gone
        console.log('[iOS Toolbar] Keyboard hidden, resetting to fixed bottom')
        toolbar.style.position = 'fixed'
        toolbar.style.top = 'auto'
        toolbar.style.bottom = '0px'
        hideToolbar(toolbar)
      }
    }

    // Listen to multiple events for iOS
    window.visualViewport.addEventListener('resize', updateToolbarPosition)
    window.visualViewport.addEventListener('scroll', updateToolbarPosition)
    window.addEventListener('scroll', updateToolbarPosition, true)

    const touchMoveHandler = () => {
      // Debounced update on scroll
      requestAnimationFrame(updateToolbarPosition)
    }
    document.addEventListener('touchmove', touchMoveHandler, { passive: true })

    // Initial positioning
    updateToolbarPosition()

    // Re-check after slight delay (iOS rendering quirk)
    setTimeout(updateToolbarPosition, 100)

    // Return cleanup function
    return () => {
      window.visualViewport.removeEventListener('resize', updateToolbarPosition)
      window.visualViewport.removeEventListener('scroll', updateToolbarPosition)
      window.removeEventListener('scroll', updateToolbarPosition, true)
      document.removeEventListener('touchmove', touchMoveHandler)
    }
  } else if (window.visualViewport) {
    // Android and other modern browsers - use simpler approach
    const updateToolbarPosition = () => {
      const viewport = window.visualViewport
      const keyboardHeight = window.innerHeight - viewport.height
      const keyboardVisible = keyboardHeight > 100

      console.log('[Toolbar Detection] Viewport change:', {
        windowHeight: window.innerHeight,
        viewportHeight: viewport.height,
        keyboardHeight,
        keyboardVisible
      })

      if (keyboardVisible) {
        toolbar.style.bottom = `${keyboardHeight}px`
        showToolbar(toolbar)
      } else {
        toolbar.style.bottom = '0px'
        hideToolbar(toolbar)
      }
    }

    window.visualViewport.addEventListener('resize', updateToolbarPosition)
    window.visualViewport.addEventListener('scroll', updateToolbarPosition)

    // Also listen to focus events as a fallback detection method
    const handleFocus = (e) => {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.classList.contains('xterm-helper-textarea')) {
        console.log('[Toolbar Detection] Focus detected, forcing show')
        // Force show toolbar when input is focused
        setTimeout(updateToolbarPosition, 100)
        setTimeout(updateToolbarPosition, 300)
      }
    }
    document.addEventListener('focusin', handleFocus)

    // Initial check
    updateToolbarPosition()

    // Re-check after delay for reliability
    setTimeout(updateToolbarPosition, 100)
    setTimeout(updateToolbarPosition, 500)

    // Return cleanup function
    return () => {
      window.visualViewport.removeEventListener('resize', updateToolbarPosition)
      window.visualViewport.removeEventListener('scroll', updateToolbarPosition)
      document.removeEventListener('focusin', handleFocus)
    }
  } else {
    // Fallback: show when input is focused
    let lastWindowHeight = window.innerHeight

    const handleFocus = (e) => {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.classList.contains('xterm-helper-textarea')) {
        showToolbar(toolbar)

        setTimeout(() => {
          const currentHeight = window.innerHeight
          const keyboardHeight = lastWindowHeight - currentHeight
          if (keyboardHeight > 100) {
            toolbar.style.bottom = `${keyboardHeight}px`
          }
        }, 300)
      }
    }

    const handleBlur = () => {
      setTimeout(() => {
        if (!document.activeElement || (document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA')) {
          toolbar.style.bottom = '0px'
          hideToolbar(toolbar)
        }
      }, 100)
    }

    const handleResize = () => {
      lastWindowHeight = window.innerHeight
    }

    document.addEventListener('focusin', handleFocus)
    document.addEventListener('focusout', handleBlur)
    window.addEventListener('resize', handleResize)

    // Return cleanup function
    return () => {
      document.removeEventListener('focusin', handleFocus)
      document.removeEventListener('focusout', handleBlur)
      window.removeEventListener('resize', handleResize)
    }
  }
}

function showToolbar(toolbar) {
  toolbar.classList.remove('hidden')
  toolbarState.visible = true
}

function hideToolbar(toolbar) {
  toolbar.classList.add('hidden')
  toolbarState.visible = false
  // Reset position when hiding
  setTimeout(() => {
    toolbar.classList.remove('ios-keyboard-mode')
    toolbar.style.top = 'auto'
    toolbar.style.bottom = '0px'
    toolbar.style.position = 'fixed'
  }, 200) // After transition completes
}

/**
 * Update toolbar UI based on state
 * When AI input has text: show context button only
 * When AI input is empty: show shortcut keys only
 */
function updateToolbarUI(contextRow, keysRow, contextModeBtn) {
  const hasAIInput = toolbarState.aiInputValue.trim().length > 0

  // Mutually exclusive visibility
  if (hasAIInput) {
    contextRow?.classList.remove('hidden')
    keysRow?.classList.add('hidden')
  } else {
    contextRow?.classList.add('hidden')
    keysRow?.classList.remove('hidden')
  }

  if (contextModeBtn) {
    contextModeBtn.classList.toggle('active', toolbarState.contextMode)
    const count = toolbarState.selectedLines.length
    contextModeBtn.innerHTML = `
      <span class="context-icon">⚡</span>
      Context (${count})
    `
  }
}

/**
 * Send quick key to terminal
 */
function sendQuickKey(key, sendKeyToTerminalFn) {
  if (key.startsWith('\\x')) {
    // Control character (e.g., \x03 for Ctrl+C)
    const charCode = parseInt(key.slice(2), 16)
    const char = String.fromCharCode(charCode)
    sendKeyToTerminalFn(char)
  } else {
    sendKeyToTerminalFn(key)
  }
}

/**
 * Toggle modifier key
 */
function toggleModifier(modifier, btn) {
  toolbarState.modifiers[modifier] = !toolbarState.modifiers[modifier]
  btn.classList.toggle('active', toolbarState.modifiers[modifier])
}

/**
 * Handle modified keypress (ctrl/alt + key)
 */
function handleModifiedKey(e, sendKeyToTerminalFn) {
  // Don't intercept if user is typing in AI input
  if (e.target.id === 'aiCommandInput') return

  const { ctrl, alt } = toolbarState.modifiers
  if (!ctrl && !alt) return

  e.preventDefault()

  let key = e.key
  if (ctrl && alt) {
    // Both modifiers
    key = `\\x${(e.keyCode).toString(16)}`
  } else if (ctrl) {
    // Ctrl+key sends control character
    if (key.length === 1) {
      const charCode = key.toUpperCase().charCodeAt(0) - 64
      key = String.fromCharCode(charCode)
    }
  }
  // Alt+key just sends escape sequence typically, but for simplicity we'll just send the key
  // In a real implementation, you'd send: \x1b + key

  sendKeyToTerminalFn(key)

  // Clear modifiers after use
  toolbarState.modifiers.ctrl = false
  toolbarState.modifiers.alt = false
  document.querySelectorAll('.modifier-key').forEach(btn => btn.classList.remove('active'))
}

/**
 * Handle AI command generation
 */
async function handleAIGenerate(aiInput, getActiveTabFn, sendKeyToTerminalFn) {
  const prompt = aiInput.value.trim()
  if (!prompt) return

  const tab = getActiveTabFn()
  if (!tab) {
    alert('No active terminal')
    return
  }

  // Show loading state
  const generateBtn = document.getElementById('aiGenerateBtn')
  const originalHTML = generateBtn.innerHTML
  generateBtn.disabled = true
  generateBtn.innerHTML = '<span class="loading-spinner"></span>'

  try {
    // Get context if in context mode
    const contextLines = toolbarState.selectedLines.map(l => l.text)

    // Call AI command generation API
    const response = await fetch('/api/generate-command', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt, context: contextLines })
    })

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`)
    }

    const data = await response.json()
    const command = data.command

    if (command) {
      // Send command to terminal
      sendKeyToTerminalFn(command)

      // Clear input
      aiInput.value = ''
      toolbarState.aiInputValue = ''
      toolbarState.contextMode = false
      toolbarState.selectedLines = []

      // Update UI
      const contextRow = document.querySelector('.mobile-toolbar-context-row')
      const keysRow = document.querySelector('.mobile-toolbar-keys-row')
      const contextModeBtn = document.getElementById('contextModeBtn')
      updateToolbarUI(contextRow, keysRow, contextModeBtn)
    }
  } catch (error) {
    console.error('AI generation error:', error)
    alert(`Failed to generate command: ${error.message}`)
  } finally {
    generateBtn.disabled = false
    generateBtn.innerHTML = originalHTML
  }
}

/**
 * Toggle context selection mode
 */
function toggleContextSelectionMode(getActiveTabFn, contextRow, keysRow, contextModeBtn) {
  const tab = getActiveTabFn()
  if (!tab || !tab.container) return

  if (toolbarState.contextMode) {
    // Enable context selection mode
    tab.container.classList.add('context-selection-mode')
    enableLineSelection(tab)
  } else {
    // Disable context selection mode
    tab.container.classList.remove('context-selection-mode')
    disableLineSelection(tab)
    // Clear selection highlights
    clearLineSelections(tab)
  }
}

/**
 * Enable line selection in terminal
 */
function enableLineSelection(tab) {
  if (!tab.term || !tab.container) return

  // We need to intercept clicks on terminal rows
  // xterm.js doesn't expose individual lines easily, so we'll use a workaround
  // by reading the terminal buffer

  const handleTerminalClick = (e) => {
    // Get click position relative to terminal
    const rect = tab.container.getBoundingClientRect()
    const y = e.clientY - rect.top

    // Estimate which line was clicked based on terminal row height
    const rowHeight = rect.height / tab.term.rows
    const lineIndex = Math.floor(y / rowHeight)

    if (lineIndex >= 0 && lineIndex < tab.term.rows) {
      toggleLineSelection(tab, lineIndex)
    }
  }

  // Store handler for cleanup
  tab._contextClickHandler = handleTerminalClick
  tab.container.addEventListener('click', handleTerminalClick)
}

/**
 * Disable line selection
 */
function disableLineSelection(tab) {
  if (tab._contextClickHandler && tab.container) {
    tab.container.removeEventListener('click', tab._contextClickHandler)
    delete tab._contextClickHandler
  }
}

/**
 * Toggle line selection
 */
function toggleLineSelection(tab, lineIndex) {
  try {
    // Get line content from terminal buffer
    const buffer = tab.term.buffer.active
    const line = buffer.getLine(lineIndex)

    if (!line) return

    // Get line text
    let lineText = ''
    for (let i = 0; i < line.length; i++) {
      const cell = line.getCell(i)
      if (cell) {
        lineText += cell.getChars()
      }
    }
    lineText = lineText.trim()

    if (!lineText) return

    // Check if line already selected
    const existingIndex = toolbarState.selectedLines.findIndex(l => l.index === lineIndex)

    if (existingIndex >= 0) {
      // Deselect
      toolbarState.selectedLines.splice(existingIndex, 1)
      removeLineHighlight(tab, lineIndex)
    } else {
      // Select
      toolbarState.selectedLines.push({ index: lineIndex, text: lineText })
      addLineHighlight(tab, lineIndex)
    }

    // Update context button
    updateContextButton()
  } catch (error) {
    console.error('Error toggling line selection:', error)
  }
}

/**
 * Add visual highlight to selected line
 */
function addLineHighlight(tab, lineIndex) {
  // Add a CSS class to highlight the line
  // This is tricky with xterm.js, so we'll use a decorations API if available
  // For now, we'll just keep track and the CSS will handle hover states
  if (!tab._selectedLineIndexes) {
    tab._selectedLineIndexes = new Set()
  }
  tab._selectedLineIndexes.add(lineIndex)
}

/**
 * Remove highlight from line
 */
function removeLineHighlight(tab, lineIndex) {
  if (tab._selectedLineIndexes) {
    tab._selectedLineIndexes.delete(lineIndex)
  }
}

/**
 * Clear all line selections
 */
function clearLineSelections(tab) {
  toolbarState.selectedLines = []
  if (tab._selectedLineIndexes) {
    tab._selectedLineIndexes.clear()
  }
  updateContextButton()
}

/**
 * Update context button label
 */
function updateContextButton() {
  const contextModeBtn = document.getElementById('contextModeBtn')
  if (!contextModeBtn) return

  const count = toolbarState.selectedLines.length
  contextModeBtn.innerHTML = `
    <span class="context-icon">⚡</span>
    Context (${count} ${count === 1 ? 'line' : 'lines'})
  `
}

/**
 * Show keyboard shortcut hints
 */
function showModifierHints(modifier, btn) {
  const hints = {
    ctrl: [
      'Ctrl+C - Interrupt',
      'Ctrl+D - EOF',
      'Ctrl+Z - Suspend',
      'Ctrl+L - Clear screen',
      'Ctrl+A - Line start',
      'Ctrl+E - Line end',
      'Ctrl+U - Clear line',
      'Ctrl+K - Kill to end',
      'Ctrl+W - Delete word',
      'Ctrl+R - Search history'
    ],
    alt: [
      'Alt+B - Back word',
      'Alt+F - Forward word',
      'Alt+D - Delete word',
      'Alt+. - Last argument'
    ]
  }

  const hintText = hints[modifier] || []
  if (hintText.length === 0) return

  // Create hint overlay
  const overlay = document.createElement('div')
  overlay.className = 'keyboard-hints-overlay'
  overlay.innerHTML = `
    <div class="keyboard-hints-content">
      <h4>${modifier.toUpperCase()} Shortcuts</h4>
      <ul>
        ${hintText.map(hint => `<li>${hint}</li>`).join('')}
      </ul>
      <p class="hint-note">Tap to close</p>
    </div>
  `

  overlay.addEventListener('click', () => {
    overlay.remove()
  })

  document.body.appendChild(overlay)

  // Animate in
  requestAnimationFrame(() => {
    overlay.style.opacity = '1'
  })

  // Auto-remove after 5 seconds
  setTimeout(() => {
    if (overlay.parentElement) {
      overlay.style.opacity = '0'
      setTimeout(() => overlay.remove(), 200)
    }
  }, 5000)
}
