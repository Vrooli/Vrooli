import React, { useRef, useEffect } from 'react'
import { Highlight, themes } from 'prism-react-renderer'

interface CodeEditorProps {
  value: string
  onChange: (value: string) => void
  language?: string
  placeholder?: string
  className?: string
}

export function CodeEditor({ value, onChange, language = 'go', placeholder, className = '' }: CodeEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const highlightRef = useRef<HTMLDivElement>(null)

  // Sync scroll between textarea and highlight
  useEffect(() => {
    const textarea = textareaRef.current
    const highlight = highlightRef.current
    
    if (!textarea || !highlight) return

    const syncScroll = () => {
      if (highlight) {
        highlight.scrollTop = textarea.scrollTop
        highlight.scrollLeft = textarea.scrollLeft
      }
    }

    textarea.addEventListener('scroll', syncScroll)
    return () => textarea.removeEventListener('scroll', syncScroll)
  }, [])

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange(e.target.value)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Tab') {
      e.preventDefault()
      const target = e.target as HTMLTextAreaElement
      const start = target.selectionStart
      const end = target.selectionEnd
      const newValue = value.substring(0, start) + '    ' + value.substring(end)
      onChange(newValue)
      
      // Set cursor position after the tab
      setTimeout(() => {
        target.selectionStart = target.selectionEnd = start + 4
      }, 0)
    }
  }

  return (
    <div className={`relative bg-gray-900 border border-gray-600 rounded-lg overflow-hidden ${className}`}>
      {/* Editable textarea - on top for input */}
      <textarea
        ref={textareaRef}
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className="absolute inset-0 w-full h-full p-4 pl-16 text-sm font-mono text-white bg-transparent border-0 outline-none resize-none z-20"
        style={{ 
          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
          lineHeight: '1.5'
        }}
        spellCheck={false}
        autoComplete="off"
        autoCorrect="off"
        autoCapitalize="off"
      />

      {/* Syntax highlighted background - below textarea */}
      <div 
        ref={highlightRef}
        className="absolute inset-0 pointer-events-none overflow-hidden z-10"
        style={{ 
          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
          opacity: value ? 1 : 0.3
        }}
      >
        <Highlight theme={themes.vsDark} code={value || placeholder || ''} language={language}>
          {({ className: highlightClassName, style, tokens, getLineProps, getTokenProps }) => (
            <pre 
              className={`${highlightClassName} text-sm h-full p-4 m-0 whitespace-pre-wrap`} 
              style={style}
            >
              <code className="block">
                {tokens.map((line, i) => (
                  <div key={i} {...getLineProps({ line, key: i })} className="table-row">
                    <span className="table-cell text-right pr-4 select-none text-gray-500 text-xs w-12" style={{ minWidth: '3ch' }}>
                      {i + 1}
                    </span>
                    <span className="table-cell">
                      {line.map((token, key) => (
                        <span key={key} {...getTokenProps({ token, key })} />
                      ))}
                      {/* Add empty space for empty lines */}
                      {line.length === 0 && <span>&nbsp;</span>}
                    </span>
                  </div>
                ))}
              </code>
            </pre>
          )}
        </Highlight>
      </div>
    </div>
  )
}