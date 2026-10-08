import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test-utils/renderWithProviders'
import { GraphHelpContent } from './GraphHelpContent'

describe('GraphHelpContent', () => {
  it('renders section navigation and merged legend content', () => {
    render(<GraphHelpContent />)

    expect(screen.getByText('Sections')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Overview' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Legend' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Queries' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Interactions' })).toBeInTheDocument()

    // Legend content is now embedded directly in help.
    expect(screen.getByText('Node Types')).toBeInTheDocument()
    expect(screen.getByText('Health')).toBeInTheDocument()
    expect(screen.getByText('Edge Types')).toBeInTheDocument()
  })

  it('renders query reference cards', () => {
    render(<GraphHelpContent />)

    expect(screen.getByText('Orphaned Skills')).toBeInTheDocument()
    expect(screen.getByText('Skillless Agents')).toBeInTheDocument()
    expect(screen.getByText('CLI-less Skills')).toBeInTheDocument()
    expect(screen.getByText('Circular Refs')).toBeInTheDocument()
  })
})
