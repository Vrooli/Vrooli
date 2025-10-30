import { useState } from 'react'
import { Lock, ExternalLink, ClipboardCopy, RefreshCw } from 'lucide-react'
import toast from 'react-hot-toast'

interface AuthPromptProps {
  loginUrl: string
}

export function AuthPrompt({ loginUrl }: AuthPromptProps) {
  const [copied, setCopied] = useState(false)

  const handleOpenLogin = () => {
    window.location.href = loginUrl
  }

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(loginUrl)
      setCopied(true)
      toast.success('Login link copied to clipboard')
      window.setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('[Calendar] Failed to copy login link', error)
      toast.error('Could not copy login link')
    }
  }

  const handleRefresh = () => {
    window.location.reload()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/85 px-4 backdrop-blur">
      <div className="w-full max-w-lg rounded-2xl bg-white p-8 shadow-2xl dark:bg-gray-900 dark:text-gray-100">
        <div className="mx-auto mb-6 flex h-14 w-14 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-200">
          <Lock className="h-7 w-7" />
        </div>
        <h2 className="mb-3 text-center text-2xl font-semibold">Sign in required</h2>
        <p className="mb-6 text-center text-sm text-gray-600 dark:text-gray-300">
          Your session is not authenticated. Sign in through the scenario authenticator to continue using the calendar.
        </p>

        <div className="mb-6 rounded-lg bg-gray-100 p-4 text-xs font-mono text-gray-700 dark:bg-gray-800 dark:text-gray-200">
          {loginUrl}
        </div>

        <div className="flex flex-col gap-3 sm:flex-row">
          <button
            onClick={handleOpenLogin}
            className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
          >
            <ExternalLink className="h-4 w-4" />
            Open Login Portal
          </button>
          <button
            onClick={handleCopyLink}
            className="flex items-center justify-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition hover:bg-gray-100 focus:outline-none dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800"
          >
            <ClipboardCopy className="h-4 w-4" />
            {copied ? 'Copied' : 'Copy Link'}
          </button>
          <button
            onClick={handleRefresh}
            className="flex items-center justify-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition hover:bg-gray-100 focus:outline-none dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh After Sign-In
          </button>
        </div>
      </div>
    </div>
  )
}
