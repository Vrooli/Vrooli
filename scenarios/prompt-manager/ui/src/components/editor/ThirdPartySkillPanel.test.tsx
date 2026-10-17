import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test-utils/renderWithProviders'
import { ThirdPartySkillPanel } from './ThirdPartySkillPanel'

describe('ThirdPartySkillPanel', () => {
  it('credits the source and names external tools Vrooli does not provide', () => {
    render(
      <ThirdPartySkillPanel
        skill={{
          id: 'brag',
          origin: {
            kind: 'imported',
            sourceUrl: 'https://github.com/latent-spaces/brag',
            commit: '1f8d9ade17d0ad4419cca9305fbc1398a4dd5b39',
            license: 'MIT',
            review: { verdict: 'passed', reviewer: 'operator' },
          },
          externalTools: [{ name: 'hyperframes', url: 'https://hyperframes.heygen.com/', purpose: 'renders the video' }],
        }}
      />,
    )
    expect(screen.getByText('Third-party skill')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /latent-spaces\/brag/ })).toHaveAttribute('href', 'https://github.com/latent-spaces/brag')
    expect(screen.getByText('License MIT')).toBeInTheDocument()
    expect(screen.getByText(/does not provide or manage/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'hyperframes' })).toHaveAttribute('href', 'https://hyperframes.heygen.com/')
  })

  it('renders nothing for authored skills', () => {
    const { container } = render(<ThirdPartySkillPanel skill={{ id: 'authored', origin: null, externalTools: [] }} />)
    expect(container).toBeEmptyDOMElement()
  })
})
