import { beforeEach, describe, expect, it } from 'vitest'
import { useSelectionStore } from './selectionStore'

describe('selectionStore', () => {
  beforeEach(() => {
    useSelectionStore.setState({ selectedSkillIds: [] })
  })

  it('starts with no selected skills', () => {
    expect(useSelectionStore.getState().selectedSkillIds).toEqual([])
  })

  it('toggles skill multi-selection', () => {
    useSelectionStore.getState().toggleSkillSelection('skill-1')
    useSelectionStore.getState().toggleSkillSelection('skill-2')
    expect(useSelectionStore.getState().selectedSkillIds).toEqual(['skill-1', 'skill-2'])

    useSelectionStore.getState().toggleSkillSelection('skill-1')
    expect(useSelectionStore.getState().selectedSkillIds).toEqual(['skill-2'])
  })

  it('adds without duplicating', () => {
    useSelectionStore.getState().addToSelection('skill-1')
    useSelectionStore.getState().addToSelection('skill-1')
    expect(useSelectionStore.getState().selectedSkillIds).toEqual(['skill-1'])
  })

  it('removes selected skills', () => {
    useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])
    useSelectionStore.getState().removeFromSelection('skill-1')
    expect(useSelectionStore.getState().selectedSkillIds).toEqual(['skill-2'])
  })

  it('clears selection', () => {
    useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])
    useSelectionStore.getState().clearAllSelection()
    expect(useSelectionStore.getState().selectedSkillIds).toEqual([])
  })
})
