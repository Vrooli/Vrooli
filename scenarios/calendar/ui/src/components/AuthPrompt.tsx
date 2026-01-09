import { FormEvent, useState } from 'react'
import { Lock, ExternalLink, ClipboardCopy, RefreshCw, Mail, KeyRound } from 'lucide-react'
import toast from 'react-hot-toast'
import { api } from '@/services/api'
import { storeAuthSession } from '@/utils/auth'
import { useCalendarStore } from '@/stores/calendarStore'
import { isAxiosError } from 'axios'

interface AuthPromptProps {
  loginUrl: string
}

export function AuthPrompt({ loginUrl }: AuthPromptProps) {
  const [copied, setCopied] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loginError, setLoginError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const setAuthRequired = useCalendarStore((state) => state.setAuthRequired)

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

  const handleInlineLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!email || !password) {
      setLoginError('Email and password are required')
      return
    }

    setIsSubmitting(true)
    setLoginError(null)
    try {
      const response = await api.login({ email, password })
      if (!response?.token) {
        throw new Error(response?.message || 'Authentication failed')
      }

      storeAuthSession({
        token: response.token,
        refreshToken: response.refresh_token || response.refreshToken || null,
        user: (response.user as unknown as Record<string, unknown>) || null
      })

      toast.success('Signed in successfully')
      setAuthRequired(false, null)
      window.setTimeout(() => window.location.reload(), 300)
    } catch (error) {
      if (isAxiosError(error)) {
        const apiMessage = (error.response?.data as { error?: string; message?: string })
        setLoginError(apiMessage?.message || apiMessage?.error || 'Invalid credentials')
      } else if (error instanceof Error) {
        setLoginError(error.message)
      } else {
        setLoginError('Login failed. Please try again.')
      }
      toast.error('Could not sign in with the provided credentials')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/85 px-4 backdrop-blur">
      <div className="w-full max-w-lg rounded-2xl bg-white p-8 shadow-2xl dark:bg-gray-900 dark:text-gray-100">
        <div className="mx-auto mb-6 flex h-14 w-14 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-200">
          <Lock className="h-7 w-7" />
        </div>
        <h2 className="mb-3 text-center text-2xl font-semibold">Sign in required</h2>
        <p className="mb-6 text-center text-sm text-gray-600 dark:text-gray-300">
          Your session is not authenticated. Use the built-in login form below or open the scenario-authenticator portal.
        </p>

        <div className="mb-6 rounded-lg bg-gray-100 p-4 text-xs font-mono text-gray-700 dark:bg-gray-800 dark:text-gray-200">
          {loginUrl}
        </div>

        <div className="mb-8 rounded-xl border border-gray-200 p-4 dark:border-gray-700">
          <p className="mb-4 text-sm font-medium text-gray-700 dark:text-gray-200">Quick sign-in</p>
          <form className="space-y-4" onSubmit={handleInlineLogin}>
            <label className="flex flex-col gap-1 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
              Email
              <div className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 focus-within:border-primary-500 focus-within:ring-1 focus-within:ring-primary-500 dark:border-gray-700">
                <Mail className="h-4 w-4 text-gray-400" />
                <input
                  type="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  className="flex-1 bg-transparent text-sm outline-none"
                  placeholder="you@example.com"
                  autoComplete="email"
                />
              </div>
            </label>

            <label className="flex flex-col gap-1 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
              Password
              <div className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 focus-within:border-primary-500 focus-within:ring-1 focus-within:ring-primary-500 dark:border-gray-700">
                <KeyRound className="h-4 w-4 text-gray-400" />
                <input
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="flex-1 bg-transparent text-sm outline-none"
                  placeholder="Enter your password"
                  autoComplete="current-password"
                />
              </div>
            </label>

            {loginError && <p className="text-sm text-red-600">{loginError}</p>}

            <button
              type="submit"
              disabled={isSubmitting}
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-700 disabled:opacity-60"
            >
              {isSubmitting ? 'Signing in…' : 'Sign in securely'}
            </button>
            <p className="text-xs text-gray-500">
              Credentials never leave this scenario. We proxy the request directly to the authenticator API.
            </p>
          </form>
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
