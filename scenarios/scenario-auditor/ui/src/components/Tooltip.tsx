import { useState, useRef, useEffect } from 'react'
import { Info } from 'lucide-react'
import clsx from 'clsx'

interface TooltipProps {
  content: string
  children?: React.ReactNode
  position?: 'top' | 'bottom' | 'left' | 'right'
  className?: string
}

export default function Tooltip({ 
  content, 
  children, 
  position = 'top',
  className 
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const [tooltipPosition, setTooltipPosition] = useState({ top: 0, left: 0 })
  const triggerRef = useRef<HTMLDivElement>(null)
  const tooltipRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isVisible && triggerRef.current && tooltipRef.current) {
      const triggerRect = triggerRef.current.getBoundingClientRect()
      const tooltipRect = tooltipRef.current.getBoundingClientRect()
      
      let top = 0
      let left = 0

      switch (position) {
        case 'top':
          top = triggerRect.top - tooltipRect.height - 8
          left = triggerRect.left + (triggerRect.width - tooltipRect.width) / 2
          break
        case 'bottom':
          top = triggerRect.bottom + 8
          left = triggerRect.left + (triggerRect.width - tooltipRect.width) / 2
          break
        case 'left':
          top = triggerRect.top + (triggerRect.height - tooltipRect.height) / 2
          left = triggerRect.left - tooltipRect.width - 8
          break
        case 'right':
          top = triggerRect.top + (triggerRect.height - tooltipRect.height) / 2
          left = triggerRect.right + 8
          break
      }

      // Keep tooltip within viewport
      const padding = 10
      if (left < padding) left = padding
      if (left + tooltipRect.width > window.innerWidth - padding) {
        left = window.innerWidth - tooltipRect.width - padding
      }
      if (top < padding) top = triggerRect.bottom + 8 // Switch to bottom if no room at top
      if (top + tooltipRect.height > window.innerHeight - padding) {
        top = triggerRect.top - tooltipRect.height - 8 // Switch to top if no room at bottom
      }

      setTooltipPosition({ top, left })
    }
  }, [isVisible, position])

  return (
    <>
      <div
        ref={triggerRef}
        className={clsx("inline-flex items-center", className)}
        onMouseEnter={() => setIsVisible(true)}
        onMouseLeave={() => setIsVisible(false)}
      >
        {children || <Info className="h-4 w-4 text-dark-400 hover:text-dark-600 cursor-help" />}
      </div>
      
      {isVisible && (
        <div
          ref={tooltipRef}
          className="fixed z-50 px-3 py-2 text-sm text-white bg-dark-900 rounded-lg shadow-lg pointer-events-none max-w-xs"
          style={{
            top: `${tooltipPosition.top}px`,
            left: `${tooltipPosition.left}px`,
          }}
        >
          {content}
          <div
            className={clsx(
              "absolute w-0 h-0 border-solid",
              position === 'top' && "bottom-[-8px] left-1/2 -translate-x-1/2 border-t-8 border-t-dark-900 border-x-8 border-x-transparent",
              position === 'bottom' && "top-[-8px] left-1/2 -translate-x-1/2 border-b-8 border-b-dark-900 border-x-8 border-x-transparent",
              position === 'left' && "right-[-8px] top-1/2 -translate-y-1/2 border-l-8 border-l-dark-900 border-y-8 border-y-transparent",
              position === 'right' && "left-[-8px] top-1/2 -translate-y-1/2 border-r-8 border-r-dark-900 border-y-8 border-y-transparent"
            )}
          />
        </div>
      )}
    </>
  )
}