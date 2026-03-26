/**
 * Clipboard utility with fallback for non-secure contexts (HTTP proxies, tunnels).
 *
 * navigator.clipboard.writeText() can fail when:
 * - Not in a secure context (HTTP proxy, tunnel without HTTPS)
 * - User activation expired (after an async API call)
 * - Permissions denied or blocked by iframe sandbox
 *
 * This utility tries the Clipboard API first, then falls back to the legacy
 * document.execCommand('copy') approach on any failure.
 */
export async function copyToClipboard(text: string): Promise<void> {
  // Try modern Clipboard API first (runtime check — clipboard may be undefined in non-secure contexts)
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- navigator.clipboard is undefined in non-secure contexts (HTTP)
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // Fall through to execCommand fallback
    }
  }

  // Fallback: works in non-secure contexts and when user activation has expired
  const textarea = document.createElement('textarea')
  textarea.value = text
  // Position off-screen to avoid visual flash
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- No modern equivalent for this fallback case
    const ok = document.execCommand('copy')
    if (!ok) {
      throw new Error('Copy to clipboard failed')
    }
  } finally {
    document.body.removeChild(textarea)
  }
}

/**
 * Copy the result of an async operation to clipboard.
 *
 * The Clipboard API requires user activation (a recent click/tap). When an async
 * operation (like an API call) runs between the click and the copy, the activation
 * expires and clipboard writes fail.
 *
 * This function uses `navigator.clipboard.write()` with a `ClipboardItem` that
 * accepts a Promise — the clipboard write is initiated synchronously within the
 * user activation window, while the actual content resolves asynchronously.
 *
 * Falls back to a deferred execCommand approach for non-secure contexts.
 */
export function copyAsyncToClipboard(contentPromise: Promise<string>): Promise<void> {
  // Modern approach: ClipboardItem accepts a Promise<Blob> for deferred content.
  // This must be called synchronously in the click handler (within user activation).
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- navigator.clipboard is undefined in non-secure contexts (HTTP)
  if (navigator.clipboard?.write && typeof ClipboardItem !== 'undefined') {
    const blobPromise = contentPromise.then(
      (text) => new Blob([text], { type: 'text/plain' })
    )
    const item = new ClipboardItem({ 'text/plain': blobPromise })
    return navigator.clipboard.write([item])
  }

  // Fallback for non-secure contexts: resolve the promise, then use execCommand.
  // This may fail if user activation has expired, but it's the best we can do
  // without the modern ClipboardItem API.
  return contentPromise.then((text) => copyToClipboard(text))
}
