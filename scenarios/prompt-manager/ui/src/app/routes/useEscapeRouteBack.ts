import { useEffect } from 'react'
import { useAppBack } from './useAppBack'

function isTextEntryTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  const tag = target.tagName.toLowerCase()
  return (
    tag === 'input' ||
    tag === 'textarea' ||
    tag === 'select' ||
    target.isContentEditable ||
    target.closest('.monaco-editor') !== null
  )
}

export function useEscapeRouteBack(enabled = true, fallbackPath?: string) {
  const goBack = useAppBack(fallbackPath)

  useEffect(() => {
    if (!enabled) return undefined

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || event.defaultPrevented) return
      if (isTextEntryTarget(event.target)) return
      event.preventDefault()
      goBack()
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [enabled, goBack])
}
