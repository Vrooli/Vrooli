import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { useRef } from 'react'
import { FilePathMenu } from './FilePathMenu'

function uncontrolledSetup() {
  return render(
    <FilePathMenu
      file="test.md"
      folder="local"
      onFileChange={vi.fn()}
      onFolderChange={vi.fn()}
    />
  )
}

describe('FilePathMenu', () => {
  describe('uncontrolled mode', () => {
    it('renders its own trigger button and toggles popover on click', () => {
      uncontrolledSetup()
      const trigger = screen.getByRole('button', { name: /Open file path menu for test\.md/ })
      expect(trigger).toBeInTheDocument()
      expect(screen.queryByText('Filename')).not.toBeInTheDocument()
      fireEvent.click(trigger)
      expect(screen.getByText('Filename')).toBeInTheDocument()
    })
  })

  describe('controlled mode', () => {
    it('renders the popover when open=true and hides trigger when hideTrigger is set', () => {
      const onOpenChange = vi.fn()
      render(
        <FilePathMenu
          file="test.md"
          folder="local"
          onFileChange={vi.fn()}
          onFolderChange={vi.fn()}
          open
          onOpenChange={onOpenChange}
          hideTrigger
        />
      )
      expect(screen.queryByRole('button', { name: /Open file path menu for/ })).not.toBeInTheDocument()
      expect(screen.getByText('Filename')).toBeInTheDocument()
    })

    it('does not render the popover when open=false', () => {
      render(
        <FilePathMenu
          file="test.md"
          folder="local"
          onFileChange={vi.fn()}
          onFolderChange={vi.fn()}
          open={false}
          onOpenChange={vi.fn()}
          hideTrigger
        />
      )
      expect(screen.queryByText('Filename')).not.toBeInTheDocument()
    })

    it('calls onOpenChange(false) when clicking outside the popover', () => {
      const onOpenChange = vi.fn()
      render(
        <div>
          <div data-testid="outside">outside</div>
          <FilePathMenu
            file="test.md"
            folder="local"
            onFileChange={vi.fn()}
            onFolderChange={vi.fn()}
            open
            onOpenChange={onOpenChange}
            hideTrigger
          />
        </div>
      )
      fireEvent.mouseDown(screen.getByTestId('outside'))
      expect(onOpenChange).toHaveBeenCalledWith(false)
    })

    it('anchors to an external ref when provided', () => {
      function Harness() {
        const ref = useRef<HTMLSpanElement>(null)
        return (
          <div>
            <span ref={ref} data-testid="anchor">anchor</span>
            <FilePathMenu
              file="test.md"
              folder="local"
              onFileChange={vi.fn()}
              onFolderChange={vi.fn()}
              open
              onOpenChange={vi.fn()}
              hideTrigger
              anchorRef={ref}
            />
          </div>
        )
      }
      render(<Harness />)
      expect(screen.getByTestId('anchor')).toBeInTheDocument()
      expect(screen.getByText('Filename')).toBeInTheDocument()
    })
  })
})
