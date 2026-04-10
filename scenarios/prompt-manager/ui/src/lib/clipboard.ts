/**
 * Clipboard utility with cascading fallbacks.
 *
 * On iOS/WebKit, clipboard API calls can hang indefinitely (promise never
 * settles) when user activation expires, the document loses focus, or when
 * browser privacy features (e.g. Brave Shields) block clipboard access.
 *
 * Fallback chain (each step tried only if the previous fails):
 *   1. navigator.clipboard.writeText()  — modern, needs activation + focus
 *   2. document.execCommand('copy')     — deprecated, iOS-quirky but no hang
 *   3. navigator.share()               — mobile share sheet (user picks "Copy")
 *   4. Reject with actionable error     — suggests checking browser settings
 */

/** Timeout for any single clipboard API call — short because hangs are silent. */
const CLIPBOARD_API_TIMEOUT_MS = 3_000

/** Race a promise against a timeout. Rejects with a descriptive error on timeout. */
function withTimeout<T>(promise: Promise<T>, ms: number, label: string): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new Error(`${label} timed out after ${ms / 1000}s`)),
      ms,
    )
    promise.then(
      (v) => { clearTimeout(timer); resolve(v) },
      (e: unknown) => { clearTimeout(timer); reject(e instanceof Error ? e : new Error(String(e))) },
    )
  })
}

/**
 * Copy via execCommand. iOS requires special handling:
 * - Use an <input> (more reliable than <textarea> on iOS WebKit)
 * - readOnly prevents the keyboard from flashing
 * - setSelectionRange instead of select() (iOS ignores select())
 * - Element must be in the viewport (off-screen elements are ignored)
 */
function execCommandCopy(text: string): boolean {
  const el = document.createElement('input')
  el.setAttribute('type', 'text')
  el.setAttribute('value', text)
  el.setAttribute('readonly', '')
  // Invisible but in-viewport — iOS ignores off-screen elements for copy
  el.style.position = 'fixed'
  el.style.top = '0'
  el.style.left = '0'
  el.style.width = '1px'
  el.style.height = '1px'
  el.style.padding = '0'
  el.style.border = 'none'
  el.style.outline = 'none'
  el.style.boxShadow = 'none'
  el.style.background = 'transparent'
  el.style.opacity = '0.01' // fully 0 may be ignored on some WebKit builds
  el.style.fontSize = '16px' // prevents iOS zoom

  document.body.appendChild(el)
  el.focus()
  el.setSelectionRange(0, text.length)

  let ok = false
  try {
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- Fallback for when Clipboard API is blocked
    ok = document.execCommand('copy')
  } catch {
    // Some browsers throw instead of returning false
  } finally {
    document.body.removeChild(el)
  }
  return ok
}

/**
 * Fallback using the Web Share API (mobile). Opens the native share sheet
 * where the user can tap "Copy" to get the text on their clipboard.
 * Returns true if share was invoked (user may still cancel).
 */
async function shareAsCopyFallback(text: string): Promise<boolean> {
  if (typeof navigator.share !== 'function') return false
  try {
    await navigator.share({ text })
    return true
  } catch {
    // User cancelled or share not available — not an error
    return false
  }
}

/**
 * Copy text to clipboard, cascading through all available methods.
 */
export async function copyToClipboard(text: string): Promise<void> {
  // 1. Modern Clipboard API with timeout
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- navigator.clipboard is undefined in non-secure contexts (HTTP)
  if (navigator.clipboard?.writeText) {
    try {
      await withTimeout(
        navigator.clipboard.writeText(text),
        CLIPBOARD_API_TIMEOUT_MS,
        'clipboard.writeText',
      )
      return
    } catch {
      // writeText hung or failed — fall through
    }
  }

  // 2. execCommand with iOS-safe selection
  if (execCommandCopy(text)) return

  // 3. Web Share API (mobile) — opens share sheet where user can tap "Copy"
  const shared = await shareAsCopyFallback(text)
  if (shared) return

  // 4. Nothing worked
  throw new Error(
    'Unable to copy — your browser may be blocking clipboard access. '
    + 'Try disabling Brave Shields (tap the lion icon) or switching to Safari.',
  )
}
