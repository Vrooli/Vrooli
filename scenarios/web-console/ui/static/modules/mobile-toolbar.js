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
  }
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
      <!-- Quick Keys Row -->
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

  // Store toolbar and keyboard detection cleanup function
  let keyboardDetectionCleanup = null

  // Handle quick keys
  toolbar.querySelectorAll('.quick-key').forEach(btn => {
    // Use touchstart/mousedown to handle action immediately before focus can change
    const handlePress = (e) => {
      e.preventDefault() // Prevent default behavior that can cause focus loss
      e.stopPropagation() // Stop event from bubbling

      const key = btn.dataset.key
      sendQuickKey(key, sendKeyToTerminalFn)

      // Refocus terminal immediately to keep keyboard open
      const activeTab = getActiveTabFn()
      if (activeTab && activeTab.term) {
        // Focus the xterm textarea directly
        const textarea = activeTab.container.querySelector('.xterm-helper-textarea')
        if (textarea) {
          textarea.focus()
        } else {
          activeTab.term.focus()
        }
      }
    }

    btn.addEventListener('touchstart', handlePress, { passive: false })
    btn.addEventListener('mousedown', handlePress)

    // Prevent click event from firing (to avoid double-trigger)
    btn.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
    })
  })

  // Handle modifier keys (toggle)
  toolbar.querySelectorAll('.modifier-key').forEach(btn => {
    // Long press to show hints, short press to toggle
    let longPressTimer = null
    let isLongPress = false

    const handleStart = (e) => {
      e.preventDefault() // Prevent focus loss
      e.stopPropagation()
      isLongPress = false

      longPressTimer = setTimeout(() => {
        isLongPress = true
        showModifierHints(btn.dataset.modifier, btn)
      }, 800)
    }

    const handleEnd = (e) => {
      e.preventDefault()
      e.stopPropagation()

      if (longPressTimer) clearTimeout(longPressTimer)

      // Only toggle if it wasn't a long press
      if (!isLongPress) {
        const modifier = btn.dataset.modifier
        toggleModifier(modifier, btn)

        // Refocus terminal to keep keyboard open
        const activeTab = getActiveTabFn()
        if (activeTab && activeTab.term) {
          const textarea = activeTab.container.querySelector('.xterm-helper-textarea')
          if (textarea) {
            textarea.focus()
          } else {
            activeTab.term.focus()
          }
        }
      }
    }

    const handleCancel = () => {
      if (longPressTimer) clearTimeout(longPressTimer)
    }

    btn.addEventListener('touchstart', handleStart, { passive: false })
    btn.addEventListener('mousedown', handleStart)
    btn.addEventListener('touchend', handleEnd, { passive: false })
    btn.addEventListener('mouseup', handleEnd)
    btn.addEventListener('touchcancel', handleCancel)
    btn.addEventListener('mouseleave', handleCancel)

    // Prevent click event from firing
    btn.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
    })
  })

  // Listen for regular keypresses to apply modifiers
  document.addEventListener('keydown', (e) => {
    if (toolbarState.modifiers.ctrl || toolbarState.modifiers.alt) {
      handleModifiedKey(e, sendKeyToTerminalFn)
    }
  })

  // Apply initial mode from window global (set during workspace load) or debug checkbox
  // Check if debug toggle checkbox is checked to determine initial state
  // Also check localStorage for persisted state
  const debugToggle = document.getElementById('debugToolbarToggle')
  const persistedDebugMode = localStorage.getItem('debugToolbarEnabled') === 'true'

  // Set checkbox to match persisted state if available
  if (debugToggle && persistedDebugMode) {
    debugToggle.checked = true
  }

  const shouldEnableDebug = debugToggle ? debugToggle.checked : persistedDebugMode
  const initialMode = window.__keyboardToolbarMode || (shouldEnableDebug ? 'floating' : 'disabled')

  // Track if debug mode is active
  let debugModeActive = shouldEnableDebug

  // Setup mode with keyboard detection
  const setupMode = (mode, forceDebug = false) => {
    // Clean up previous keyboard detection
    if (keyboardDetectionCleanup) {
      keyboardDetectionCleanup()
      keyboardDetectionCleanup = null
    }

    // Update mode
    toolbarState.mode = mode

    // Track debug mode
    if (forceDebug) {
      debugModeActive = true
    }

    // Remove all mode-related classes
    toolbar.classList.remove('mode-disabled', 'mode-floating', 'mode-top')

    // Desktop mode: Hide toolbar by default unless debug mode is active
    if (!isMobile && !debugModeActive) {
      // On desktop, toolbar should be disabled by default
      toolbar.classList.add('mode-disabled', 'hidden')
      console.log(`Toolbar mode set to: disabled (desktop - use debug toggle to test)`)
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
    setMode: (mode, forceDebug = false) => {
      if (forceDebug) {
        debugModeActive = true
      } else if (mode === 'disabled') {
        debugModeActive = false
      }
      setupMode(mode, forceDebug)
    }
  }
}

/**
 * Setup keyboard visibility detection
 * iOS Safari has unique viewport behavior that requires special handling
 *
 * Key insights:
 * - iOS doesn't support 'interactive-widget=resizes-content'
 * - visualViewport is the best API for keyboard detection
 * - Position fixed with env(safe-area-inset-bottom) is most reliable
 *
 * Returns a cleanup function to remove event listeners
 */
function setupKeyboardDetection(toolbar) {

  // Detect iOS
  const isIOS = /iPhone|iPad|iPod/.test(navigator.userAgent) ||
                (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)

  if (window.visualViewport && isIOS) {
    // iOS-specific handling
    console.log('[iOS Toolbar] Initializing iOS keyboard detection')

    // Keep position fixed for smoother behavior
    toolbar.style.position = 'fixed'

    const updateToolbarPosition = () => {
      const viewport = window.visualViewport
      const keyboardHeight = window.innerHeight - viewport.height
      const keyboardVisible = keyboardHeight > 100

      // Enhanced debug logging
      console.log('[iOS Toolbar] Update:', {
        windowHeight: window.innerHeight,
        windowWidth: window.innerWidth,
        viewportHeight: viewport.height,
        viewportWidth: viewport.width,
        viewportOffsetTop: viewport.offsetTop,
        viewportOffsetLeft: viewport.offsetLeft,
        keyboardHeight,
        keyboardVisible,
        toolbarVisible: !toolbar.classList.contains('hidden'),
        toolbarZIndex: window.getComputedStyle(toolbar).zIndex,
        toolbarBottom: window.getComputedStyle(toolbar).bottom,
        toolbarPosition: window.getComputedStyle(toolbar).position
      })

      if (keyboardVisible) {
        // Position toolbar just above the keyboard
        // Use bottom positioning for more reliable results
        toolbar.style.bottom = `${keyboardHeight}px`
        toolbar.style.top = 'auto'

        // Ensure highest z-index
        toolbar.style.zIndex = '2147483647' // Max z-index value

        console.log('[iOS Toolbar] Positioning above keyboard:', {
          bottom: `${keyboardHeight}px`,
          zIndex: toolbar.style.zIndex
        })

        showToolbar(toolbar)
      } else {
        // Reset to bottom when keyboard is gone
        console.log('[iOS Toolbar] Keyboard hidden, resetting to bottom')
        toolbar.style.bottom = '0px'
        toolbar.style.top = 'auto'
        toolbar.style.zIndex = '9999'
        hideToolbar(toolbar)
      }
    }

    // Listen to multiple events for iOS
    window.visualViewport.addEventListener('resize', updateToolbarPosition)
    window.visualViewport.addEventListener('scroll', updateToolbarPosition)

    // Also listen to focus events for immediate response
    const handleFocus = (e) => {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.classList.contains('xterm-helper-textarea')) {
        console.log('[iOS Toolbar] Input focused, checking keyboard state')
        setTimeout(updateToolbarPosition, 100)
        setTimeout(updateToolbarPosition, 300)
        setTimeout(updateToolbarPosition, 500)
      }
    }
    document.addEventListener('focusin', handleFocus)

    // Initial positioning
    updateToolbarPosition()

    // Multiple checks for iOS rendering quirks
    setTimeout(updateToolbarPosition, 100)
    setTimeout(updateToolbarPosition, 300)
    setTimeout(updateToolbarPosition, 500)

    // Return cleanup function
    return () => {
      window.visualViewport.removeEventListener('resize', updateToolbarPosition)
      window.visualViewport.removeEventListener('scroll', updateToolbarPosition)
      document.removeEventListener('focusin', handleFocus)
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
    toolbar.style.top = 'auto'
    toolbar.style.bottom = '0px'
    toolbar.style.position = 'fixed'
  }, 200) // After transition completes
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
