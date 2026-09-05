/** Simulation behaviour settings remain an enforced gate during visual migration. */
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { literalAllowlists, scanLiterals } from '../../__lint__/literals'

describe('sim has no behaviour literals', () => {
  it('every numeric literal outside algorithm exemptions is structural', () => {
    expect(scanLiterals(resolve(import.meta.dirname, '..'), literalAllowlists.sim)).toEqual([])
  })
})
