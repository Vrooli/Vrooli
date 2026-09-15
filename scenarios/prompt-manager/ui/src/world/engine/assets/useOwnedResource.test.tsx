import { StrictMode } from 'react'
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useOwnedResource } from './useOwnedResource'

describe('committed resource ownership', () => {
  it('disposes every allocation once through StrictMode replay, replacement and unmount', () => {
    const resources: Array<{ dispose: ReturnType<typeof vi.fn> }> = []
    const first = () => { const resource = { dispose: vi.fn() }; resources.push(resource); return resource }
    const second = () => { const resource = { dispose: vi.fn() }; resources.push(resource); return resource }
    const view = renderHook(({ factory }) => useOwnedResource(factory), { initialProps: { factory: first }, wrapper: StrictMode })
    expect(view.result.current?.dispose).not.toHaveBeenCalled()
    view.rerender({ factory: second })
    const current = view.result.current
    expect(current?.dispose).not.toHaveBeenCalled()
    for (const resource of resources) if (resource !== current) expect(resource.dispose).toHaveBeenCalledOnce()
    view.unmount()
    for (const resource of resources) expect(resource.dispose).toHaveBeenCalledOnce()
  })
})
