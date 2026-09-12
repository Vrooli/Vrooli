import { act, fireEvent, render, screen, within } from '@/test-utils/renderWithProviders'
import { describe, expect, it, vi } from 'vitest'
import { ConnectedSpaceMenu, SpaceMenu } from './TeamPanel'
import { makeWorld, makeWorldStore } from '../sim/__tests__/fixtures'

describe('space menu', () => {
  it('updates shared-room occupants on simulation ticks without stealing focus', () => {
    const store = makeWorldStore({ scene: 'office', teams: 1, agents: 1 })
    render(<ConnectedSpaceMenu store={store} spaceId="shared:lounge" walking onManage={() => {}} onAgent={() => {}} onClose={() => {}} />)
    expect(screen.getByText('No agents here or assigned to this space.')).toBeVisible()
    const manage = screen.getByRole('button', { name: 'Manage space' })
    manage.focus()
    const state = store.getState(), actor = state.actors[state.actorOrder[0] ?? ''], lounge = state.places['shared:lounge']
    if (!actor || !lounge) throw new Error('Missing fixture')
    act(() => {
      actor.position = [...lounge.position]
      actor.idle.until = Infinity
      store.advance(store.tuning().sim.tickSeconds)
    })
    expect(screen.getByText(actor.name)).toBeVisible()
    expect(screen.getByRole('button', { name: 'Come here' })).toBeVisible()
    expect(manage).toHaveFocus()
  })
  it.each([false, true])('exposes the enclosure and its occupants with walking=%s', walking => {
    const world = makeWorld({ teams: 1, agents: 2 })
    const space = world.places['room:team-0']
    if (!space) throw new Error('Missing fixture space')
    const onManage = vi.fn(), onAgent = vi.fn(), onClose = vi.fn()
    const actors = Object.values(world.actors)
    render(<SpaceMenu space={space} actors={actors} walking={walking} onManage={onManage} onAgent={onAgent} onClose={onClose} />)
    const dialog = screen.getByRole('dialog', { name: 'Team 0 space' })
    expect(within(dialog).getByRole('button', { name: 'Close space menu' })).toHaveFocus()
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
    onClose.mockClear()
    expect(within(dialog).getByRole('list', { name: 'Space occupants' })).toBeVisible()
    fireEvent.click(within(dialog).getAllByRole('button', { name: walking ? 'Come here' : 'View agent' })[0] as HTMLElement)
    expect(onAgent).toHaveBeenCalledWith(actors[0]?.id)
    fireEvent.click(within(dialog).getByRole('button', { name: 'Manage space' }))
    expect(onManage).toHaveBeenCalledOnce()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Close space menu' }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
