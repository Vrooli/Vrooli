import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test-utils/renderWithProviders'
import { expectNoA11yViolations } from '@/test-utils/a11y'

describe('Prompt Manager app shell accessibility', () => {
  it('has no automated axe violations in its baseline shell', async () => {
    const { container } = renderWithProviders(
      <main aria-label="Prompt Manager">
        <h1>Prompt Manager</h1>
        <nav aria-label="Primary">
          <a href="/">Skills</a>
          <a href="/graph">Graph</a>
        </nav>
      </main>,
    )

    expect(screen.getByRole('heading', { name: 'Prompt Manager' })).toBeVisible()
    await expectNoA11yViolations(container)
  })
})
