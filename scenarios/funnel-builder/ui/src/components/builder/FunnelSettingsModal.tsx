import { FormEvent, useEffect, useState, useRef } from 'react'
import clsx from 'clsx'
import { Check, Info, Monitor, Smartphone, X } from 'lucide-react'
import { FunnelSettings } from '../../types'
import ResponsiveDialog from '../dialog/ResponsiveDialog'
import {
  DEFAULT_BORDER_RADIUS,
  DEFAULT_PRIMARY_COLOR,
  darkenColor,
  getFontStack as resolveFontStack,
  lightenColor,
  sanitizeHexColor
} from '../../utils/theme'

interface FunnelSettingsModalProps {
  settings: FunnelSettings
  onClose: () => void
  onSave: (settings: FunnelSettings, scope: 'funnel' | 'project') => void
  canApplyToProject?: boolean
}

interface FontOption {
  value: string
  label: string
  stack: string
}

const fontOptions: FontOption[] = [
  {
    value: 'Inter',
    label: 'Inter',
    stack: "'Inter', 'Inter var', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
  },
  {
    value: 'Poppins',
    label: 'Poppins',
    stack: "'Poppins', 'Helvetica Neue', Helvetica, Arial, sans-serif"
  },
  {
    value: 'Rubik',
    label: 'Rubik',
    stack: "'Rubik', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif"
  },
  {
    value: 'DM Sans',
    label: 'DM Sans',
    stack: "'DM Sans', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif"
  },
  {
    value: 'IBM Plex Sans',
    label: 'IBM Plex Sans',
    stack: "'IBM Plex Sans', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif"
  }
]
const radiusOptions = [
  { value: '0.25rem', label: 'Compact' },
  { value: '0.5rem', label: 'Rounded' },
  { value: '0.75rem', label: 'Soft' },
  { value: '1rem', label: 'Pill' },
]

const defaultTheme = {
  primaryColor: DEFAULT_PRIMARY_COLOR,
  fontFamily: fontOptions[0].value,
  borderRadius: radiusOptions.find((option) => option.value === DEFAULT_BORDER_RADIUS)?.value
    ?? radiusOptions[1].value,
}

const normalizeSettings = (settings: FunnelSettings): FunnelSettings => ({
  ...settings,
  theme: {
    ...defaultTheme,
    ...(settings.theme ?? {}),
  },
  tracking: {
    ...(settings.tracking ?? {}),
  },
  progressBar: settings.progressBar ?? false,
  exitIntent: settings.exitIntent ?? false,
})

const FunnelSettingsModal = ({ settings, onClose, onSave, canApplyToProject = false }: FunnelSettingsModalProps) => {
  const [localSettings, setLocalSettings] = useState<FunnelSettings>(normalizeSettings(settings))
  const [isTrackingInfoOpen, setIsTrackingInfoOpen] = useState(false)
  const trackingInfoRef = useRef<HTMLDivElement | null>(null)
  const [previewDevice, setPreviewDevice] = useState<'desktop' | 'mobile'>('desktop')
  const [applyScope, setApplyScope] = useState<'funnel' | 'project'>(canApplyToProject ? 'funnel' : 'funnel')

  useEffect(() => {
    setLocalSettings(normalizeSettings(settings))
    setApplyScope('funnel')
  }, [settings, canApplyToProject])

  useEffect(() => {
    if (!isTrackingInfoOpen) {
      return
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (trackingInfoRef.current && !trackingInfoRef.current.contains(event.target as Node)) {
        setIsTrackingInfoOpen(false)
      }
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsTrackingInfoOpen(false)
      }
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [isTrackingInfoOpen])

  const getFontStack = (value: string): string => {
    const match = fontOptions.find((option) => option.value === value)
    if (match) {
      return match.stack
    }
    return resolveFontStack(value)
  }

  const updateTheme = (key: 'primaryColor' | 'fontFamily' | 'borderRadius', value: string) => {
    setLocalSettings((prev) => ({
      ...prev,
      theme: {
        ...(prev.theme ?? defaultTheme),
        [key]: value,
      },
    }))
  }

  const updateCheckbox = (key: 'progressBar' | 'exitIntent', value: boolean) => {
    setLocalSettings((prev) => ({
      ...prev,
      [key]: value,
    }))
  }

  const updateTracking = (key: 'googleAnalytics' | 'facebookPixel', value: string) => {
    setLocalSettings((prev) => ({
      ...prev,
      tracking: {
        ...(prev.tracking ?? {}),
        [key]: value,
      },
    }))
  }

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const scope = canApplyToProject ? applyScope : 'funnel'
    onSave(localSettings, scope)
  }

  const selectedFontFamily = localSettings.theme.fontFamily ?? fontOptions[0].value
  const previewPrimaryColor = sanitizeHexColor(localSettings.theme.primaryColor)
  const previewFontFamily = getFontStack(selectedFontFamily)
  const previewBorderRadius = localSettings.theme.borderRadius ?? radiusOptions[1].value
  const previewBackground = lightenColor(previewPrimaryColor, 0.82)
  const previewGradientEnd = darkenColor(previewPrimaryColor, 0.08)
  const previewBadgeBackground = lightenColor(previewPrimaryColor, 0.6)
  const previewBadgeText = darkenColor(previewPrimaryColor, 0.4)
  const previewProgressValue = localSettings.progressBar ? 64 : 0
  const previewWrapperClass = clsx(
    'relative overflow-hidden border border-gray-200 bg-gray-50 shadow-sm transition-all duration-300',
    previewDevice === 'mobile' ? 'mx-auto w-full max-w-xs min-h-[520px]' : 'w-full min-h-[420px]'
  )
  const previewContentPaddingClass = clsx(
    'space-y-6',
    previewDevice === 'mobile' ? 'px-4 py-5' : 'px-6 py-7'
  )

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabel="Funnel settings"
      size="xl"
      className="shadow-2xl"
    >
      <form onSubmit={handleSubmit} className="flex flex-1 min-h-0 flex-col">
        <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Funnel Settings</h3>
            <p className="text-sm text-gray-500">
              Customize how your funnel looks, behaves, and tracks performance.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="icon-btn icon-btn-ghost h-9 w-9"
            aria-label="Close settings"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-6 min-h-0">
          <div className="flex flex-col gap-10 lg:flex-row">
            <div className="w-full space-y-6 lg:w-1/3 lg:min-w-[320px]">
              <section className="space-y-4">
                <h4 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Appearance</h4>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Primary color</label>
                  <div className="mt-2 flex items-center gap-3">
                    <input
                      type="color"
                      value={localSettings.theme.primaryColor}
                      onChange={(event) => updateTheme('primaryColor', event.target.value)}
                      className="h-10 w-14 cursor-pointer rounded-md border border-gray-200 bg-white"
                      aria-label="Primary color"
                    />
                    <input
                      type="text"
                      value={localSettings.theme.primaryColor}
                      onChange={(event) => updateTheme('primaryColor', event.target.value)}
                      className="input"
                      placeholder="#2563eb"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Font family</label>
                  <div
                    role="radiogroup"
                    aria-label="Font family"
                    className="mt-2 grid gap-2"
                  >
                    {fontOptions.map((font) => {
                      const isSelected = selectedFontFamily === font.value
                      return (
                        <button
                          key={font.value}
                          type="button"
                          onClick={() => updateTheme('fontFamily', font.value)}
                          className={clsx(
                            'w-full rounded-lg border px-4 py-3 text-left transition focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2',
                            isSelected
                              ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm'
                              : 'border-gray-200 bg-white text-gray-900 hover:border-gray-300'
                          )}
                          role="radio"
                          aria-checked={isSelected}
                        >
                          <div className="flex items-center justify-between gap-3">
                            <span className="text-sm font-semibold" style={{ fontFamily: font.stack }}>
                              {font.label}
                            </span>
                            {isSelected && <Check className="h-4 w-4 text-primary-600" aria-hidden="true" />}
                          </div>
                          <span className="mt-1 block text-xs text-gray-500" style={{ fontFamily: font.stack }}>
                            The quick brown fox jumps over the lazy dog.
                          </span>
                        </button>
                      )
                    })}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Border radius</label>
                  <select
                    value={localSettings.theme.borderRadius ?? radiusOptions[1].value}
                    onChange={(event) => updateTheme('borderRadius', event.target.value)}
                    className="input"
                  >
                    {radiusOptions.map((radius) => (
                      <option key={radius.value} value={radius.value}>
                        {radius.label}
                      </option>
                    ))}
                  </select>
                </div>
              </section>

              <section className="space-y-4">
                <h4 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Behavior</h4>

                <label className="flex items-center justify-between gap-4 rounded-lg border border-gray-200 px-4 py-3">
                  <div>
                    <span className="text-sm font-medium text-gray-900">Show progress bar</span>
                    <p className="text-xs text-gray-500">Display a completion indicator across funnel steps.</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={localSettings.progressBar ?? false}
                    onChange={(event) => updateCheckbox('progressBar', event.target.checked)}
                    className="h-5 w-5"
                  />
                </label>

                <label className="flex items-center justify-between gap-4 rounded-lg border border-gray-200 px-4 py-3">
                  <div>
                    <span className="text-sm font-medium text-gray-900">Enable exit intent</span>
                    <p className="text-xs text-gray-500">Trigger a modal when visitors move to close the tab.</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={localSettings.exitIntent ?? false}
                    onChange={(event) => updateCheckbox('exitIntent', event.target.checked)}
                    className="h-5 w-5"
                  />
                </label>

                <div className="space-y-3">
                  <div className="flex items-start justify-between gap-2">
                    <h5 className="text-sm font-medium text-gray-900">Tracking</h5>
                    <div className="relative" ref={trackingInfoRef}>
                      <button
                        type="button"
                        onClick={() => setIsTrackingInfoOpen((open) => !open)}
                        className="icon-btn icon-btn-ghost h-8 w-8 text-gray-500"
                        aria-expanded={isTrackingInfoOpen}
                        aria-controls="tracking-info-popover"
                        aria-label="Learn how tracking integrations work"
                      >
                        <Info className="h-4 w-4" aria-hidden="true" />
                      </button>
                      {isTrackingInfoOpen && (
                        <div
                          id="tracking-info-popover"
                          role="dialog"
                          className="absolute right-0 top-9 z-20 w-72 rounded-lg border border-gray-200 bg-white p-4 text-xs text-gray-600 shadow-xl"
                        >
                          <p className="font-semibold text-gray-900">Why add tracking?</p>
                          <p className="mt-2">
                            Connecting Google Analytics or Facebook Pixel lets you attribute funnel performance and
                            retarget visitors who abandon the flow.
                          </p>
                          <p className="mt-2">
                            Your tracking IDs stay private to this workspace and are only injected into published funnels.
                          </p>
                        </div>
                      )}
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-gray-500">
                      Google Analytics ID
                    </label>
                    <input
                      type="text"
                      value={localSettings.tracking?.googleAnalytics ?? ''}
                      onChange={(event) => updateTracking('googleAnalytics', event.target.value)}
                      className="input mt-1"
                      placeholder="G-XXXXXXXX"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-gray-500">
                      Facebook Pixel ID
                    </label>
                    <input
                      type="text"
                      value={localSettings.tracking?.facebookPixel ?? ''}
                      onChange={(event) => updateTracking('facebookPixel', event.target.value)}
                      className="input mt-1"
                      placeholder="1234567890"
                    />
                  </div>
                </div>
              </section>
            </div>

            <section className="w-full space-y-4 lg:w-2/3">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <h4 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Live Preview</h4>
                <div className="inline-flex items-center gap-1 rounded-full border border-gray-200 bg-white p-1">
                  {[
                    { id: 'desktop', label: 'Desktop', Icon: Monitor },
                    { id: 'mobile', label: 'Mobile', Icon: Smartphone },
                  ].map(({ id, label, Icon }) => (
                    <button
                      key={id}
                      type="button"
                      onClick={() => setPreviewDevice(id as 'desktop' | 'mobile')}
                      className={clsx(
                        'flex items-center gap-2 rounded-full px-3 py-1.5 text-sm font-medium transition',
                        previewDevice === id
                          ? 'bg-indigo-600 text-white shadow'
                          : 'text-gray-600 hover:text-gray-900'
                      )}
                      aria-pressed={previewDevice === id}
                    >
                      <Icon className="h-4 w-4" />
                      {label}
                    </button>
                  ))}
                </div>
              </div>

              <div
                className={previewWrapperClass}
                style={{
                  borderRadius: previewBorderRadius,
                  fontFamily: previewFontFamily,
                  background: `linear-gradient(135deg, ${previewBackground}, ${previewGradientEnd})`
                }}
                aria-label="Funnel theme preview"
              >
                {localSettings.progressBar ? (
                  <div className="h-1 w-full bg-white/40">
                    <div
                      className="h-full rounded-r-full"
                      style={{
                        width: `${previewProgressValue}%`,
                        backgroundColor: previewPrimaryColor,
                        borderRadius: previewBorderRadius
                      }}
                    />
                  </div>
                ) : (
                  <div className="h-1 w-full bg-white/20" />
                )}

                <div className={previewContentPaddingClass}>
                  <div className="inline-flex items-center gap-2 rounded-full bg-white/70 px-3 py-1 text-xs font-semibold">
                    <span className="h-2 w-2 rounded-full" style={{ backgroundColor: previewPrimaryColor }} />
                    Funnel Preview
                  </div>

                  <div className="space-y-3 text-white">
                    <p className="text-2xl font-semibold leading-tight">
                      Guide visitors through a polished, on-brand journey.
                    </p>
                    <p className="text-sm text-white/80">
                      Fonts, colors, and components update in real-time as you tweak settings so you can
                      see the customer experience before publishing.
                    </p>
                  </div>

                  <div
                    className={clsx(
                      'rounded-2xl bg-white/95 p-5 shadow-xl backdrop-blur',
                      previewDevice === 'mobile' ? 'px-4 py-4'
                        : null
                    )}
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className="flex h-10 w-10 shrink-0 items-center justify-center text-lg font-semibold"
                        style={{
                          backgroundColor: previewBadgeBackground,
                          color: previewBadgeText,
                          borderRadius: previewBorderRadius
                        }}
                      >
                        01
                      </div>
                      <div className="space-y-1">
                        <p className="text-sm font-semibold text-gray-900">Step title</p>
                        <p className="text-xs text-gray-500">Lead capture form with tailored messaging</p>
                      </div>
                    </div>

                    <div className="mt-4 space-y-3">
                      <div className="h-10 w-full rounded-md border border-gray-200 bg-gray-50" style={{ borderRadius: previewBorderRadius }} />
                      <div className="h-10 w-full rounded-md border border-gray-200 bg-gray-50" style={{ borderRadius: previewBorderRadius }} />
                    </div>

                    <button
                      type="button"
                      className="mt-5 w-full rounded-md px-4 py-2 text-sm font-semibold text-white shadow transition-colors"
                      style={{
                        backgroundColor: previewPrimaryColor,
                        borderRadius: previewBorderRadius
                      }}
                      tabIndex={-1}
                      aria-hidden="true"
                    >
                      Primary action
                    </button>
                  </div>
                </div>

                {localSettings.exitIntent && (
                  <div
                    className={clsx(
                      'pointer-events-none absolute',
                      previewDevice === 'mobile' ? 'inset-x-4 bottom-4' : 'inset-x-6 bottom-6'
                    )}
                  >
                    <div
                      className="rounded-xl border border-gray-200 bg-white/95 p-4 text-xs text-gray-700 shadow-2xl backdrop-blur"
                      style={{ borderRadius: previewBorderRadius }}
                    >
                      <p className="font-semibold text-gray-900">Exit intent offer</p>
                      <p className="mt-1 leading-relaxed">
                        A focused modal highlights a final incentive just before your visitors close the tab.
                      </p>
                      <div className="mt-3 flex items-center gap-2 text-[11px] text-gray-500">
                        <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: previewPrimaryColor }} />
                        Triggered when the cursor leaves the viewport.
                      </div>
                    </div>
                  </div>
                )}
              </div>
              <p className="text-xs text-gray-500">
                Preview updates instantly as you adjust settings. Final content is powered by your funnel steps and lead
                forms.
              </p>
            </section>
          </div>
        </div>

        <div className="space-y-3 border-t border-gray-200 bg-gray-50 px-6 py-4">
          {canApplyToProject && (
            <fieldset className="flex flex-col gap-2">
              <legend className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                Apply changes to
              </legend>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input
                    type="radio"
                    name="funnel-settings-scope"
                    value="funnel"
                    checked={applyScope === 'funnel'}
                    onChange={() => setApplyScope('funnel')}
                  />
                  This funnel only
                </label>
                <label className={clsx(
                  'flex items-center gap-2 text-sm',
                  canApplyToProject ? 'text-gray-700' : 'text-gray-400'
                )}>
                  <input
                    type="radio"
                    name="funnel-settings-scope"
                    value="project"
                    checked={applyScope === 'project'}
                    onChange={() => setApplyScope('project')}
                    disabled={!canApplyToProject}
                  />
                  Apply to every funnel in this project
                </label>
              </div>
              <p className="text-xs text-gray-500">
                Use the project-wide option to keep design consistent across every funnel under the same project.
              </p>
            </fieldset>
          )}
          <div className="flex justify-end gap-3">
            <button type="button" onClick={onClose} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              Save Settings
            </button>
          </div>
        </div>
      </form>
    </ResponsiveDialog>
  )
}

export default FunnelSettingsModal
