/**
 * SettingsOverlay - one responsive settings surface with grouped navigation.
 *
 * Desktop uses the adopted NavigationTree as a side navigation; narrow screens
 * use the adopted Tabs strip. Both drive the same selection state, so a layout
 * change never remounts the form or duplicates it. The caller owns the form
 * state and supplies the active group's content through `renderSection`.
 */

import { useState, type ReactNode } from 'react'
import { NavigationTree } from '@vrooli/react-component-library/NavigationTree/1'
import { Tabs } from '@vrooli/react-component-library/Tabs/1'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { describeSaveState, type WorldPreferencesSaveState } from '../data/config'
import type { WorldSettingsSection } from './settingsGroups'

export interface SettingsSaveState extends WorldPreferencesSaveState {
  /** Re-attempt the last failed save. Edits are preserved either way. */
  onRetry?: () => void
}

export interface SettingsOverlayProps {
  title: string
  sections: ReadonlyArray<WorldSettingsSection>
  renderSection: (id: string) => ReactNode
  defaultSection?: string
  testId?: string
  /** Persistence state for the shared form; omitted when nothing is persisted. */
  saveState?: SettingsSaveState
}

const FAILED_SAVE_STATUSES = new Set(['error', 'unavailable', 'conflict'])

export function SettingsOverlay({
  title,
  sections,
  renderSection,
  defaultSection,
  testId = 'world-settings-overlay',
  saveState,
}: SettingsOverlayProps) {
  const isMobile = useIsMobile()
  const first = sections[0]?.id ?? ''
  const initial = sections.some((section) => section.id === defaultSection) ? (defaultSection as string) : first
  const [selected, setSelected] = useState(initial)
  const active = sections.some((section) => section.id === selected) ? selected : initial
  const failed = saveState ? FAILED_SAVE_STATUSES.has(saveState.status) : false

  return (
    <div className="space-y-3" data-testid={testId} data-active-section={active}>
      {saveState && saveState.status !== 'idle' && (
        <div
          role={failed ? 'alert' : 'status'}
          aria-live="polite"
          data-status={saveState.status}
          data-testid={`${testId}-save-status`}
          className={
            failed
              ? 'flex items-center justify-between gap-2 rounded-md border border-amber-500/60 bg-amber-500/10 px-2.5 py-1.5 text-xs text-foreground'
              : 'rounded-md px-2.5 py-1.5 text-xs text-muted-foreground'
          }
        >
          <span>{describeSaveState(saveState)}</span>
          {failed && saveState.onRetry && (
            <button
              type="button"
              onClick={saveState.onRetry}
              data-testid={`${testId}-save-retry`}
              className="rounded-md border border-border px-2 py-0.5 font-medium text-foreground hover:bg-muted"
            >
              Retry
            </button>
          )}
        </div>
      )}
      {isMobile ? (
        <Tabs
          items={sections.map((section) => ({ id: section.id, label: section.label }))}
          active={active}
          onChange={setSelected}
          ariaLabel={`${title} sections`}
          density="compact"
          testId={`${testId}-tabs`}
        />
      ) : (
        <NavigationTree title={title}>
          <ul data-rcl-navigation-tree-list>
            {sections.map((section) => {
              const isActive = section.id === active
              return (
                <li data-rcl-navigation-tree-item key={section.id}>
                  <button
                    type="button"
                    aria-current={isActive ? 'page' : undefined}
                    onClick={() => setSelected(section.id)}
                    data-testid={`${testId}-nav-${section.id}`}
                    className={
                      isActive
                        ? 'flex w-full items-center gap-2 rounded-md bg-muted px-2 py-1.5 text-left text-xs font-semibold text-foreground'
                        : 'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-muted-foreground hover:bg-muted hover:text-foreground'
                    }
                  >
                    {section.label}
                  </button>
                </li>
              )
            })}
          </ul>
        </NavigationTree>
      )}
      <div data-testid={`${testId}-section`} data-section={active}>
        {renderSection(active)}
      </div>
    </div>
  )
}
