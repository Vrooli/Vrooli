import React, { useEffect, useCallback, useState } from 'react'
import { useSymbolSearch } from '../hooks/useSymbolSearch'

/**
 * Character detail modal component
 * Shows comprehensive character information and usage examples
 */
function CharacterModal({ character, onClose }) {
  const [detailedCharacter, setDetailedCharacter] = useState(character)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const { getCharacterDetail } = useSymbolSearch()

  // Load detailed character information
  useEffect(() => {
    let isMounted = true

    async function loadDetails() {
      if (!character?.codepoint) return

      setLoading(true)
      setError(null)

      try {
        const detail = await getCharacterDetail(character.codepoint)
        if (isMounted) {
          setDetailedCharacter(detail.character)
        }
      } catch (err) {
        if (isMounted) {
          setError(err.message)
          // Use the basic character info as fallback
          setDetailedCharacter(character)
        }
      } finally {
        if (isMounted) {
          setLoading(false)
        }
      }
    }

    loadDetails()

    return () => {
      isMounted = false
    }
  }, [character, getCharacterDetail])

  // Handle keyboard events
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  // Handle backdrop click
  const handleBackdropClick = useCallback((e) => {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }, [onClose])

  // Copy text to clipboard
  const copyToClipboard = useCallback(async (text, label) => {
    try {
      await navigator.clipboard.writeText(text)
      // You could add a toast notification here
      console.log(`Copied ${label}: ${text}`)
    } catch (err) {
      console.error('Failed to copy:', err)
      // Fallback for older browsers
      const textArea = document.createElement('textarea')
      textArea.value = text
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
    }
  }, [])

  // Render character symbol safely
  const renderSymbol = useCallback(() => {
    if (!detailedCharacter) return '□'
    
    try {
      if (detailedCharacter.decimal < 32 && detailedCharacter.decimal !== 10 && detailedCharacter.decimal !== 13) {
        return '□' // Control character placeholder
      }
      return String.fromCodePoint(detailedCharacter.decimal)
    } catch (error) {
      return '□'
    }
  }, [detailedCharacter])

  // Generate usage examples
  const generateUsageExamples = useCallback(() => {
    if (!detailedCharacter) return []

    const examples = []
    
    // Unicode representation
    examples.push({
      label: 'Unicode',
      value: detailedCharacter.codepoint,
      description: 'Unicode codepoint notation'
    })

    // Decimal representation  
    examples.push({
      label: 'Decimal',
      value: detailedCharacter.decimal.toString(),
      description: 'Decimal codepoint value'
    })

    // HTML entity (decimal)
    examples.push({
      label: 'HTML Entity',
      value: `&#${detailedCharacter.decimal};`,
      description: 'HTML numeric entity'
    })

    // HTML entity (hex)
    examples.push({
      label: 'HTML Hex',
      value: `&#x${detailedCharacter.decimal.toString(16).toUpperCase()};`,
      description: 'HTML hexadecimal entity'
    })

    // CSS content
    examples.push({
      label: 'CSS Content',
      value: `\\${detailedCharacter.decimal.toString(16).toUpperCase().padStart(4, '0')}`,
      description: 'CSS content property value'
    })

    // JavaScript
    examples.push({
      label: 'JavaScript',
      value: `String.fromCodePoint(${detailedCharacter.decimal})`,
      description: 'JavaScript string generation'
    })

    // URL encoding
    const urlEncoded = encodeURIComponent(renderSymbol())
    if (urlEncoded !== renderSymbol()) {
      examples.push({
        label: 'URL Encoded',
        value: urlEncoded,
        description: 'URL percent encoding'
      })
    }

    return examples
  }, [detailedCharacter, renderSymbol])

  const usageExamples = generateUsageExamples()

  if (!character) return null

  return (
    <div className="modal-overlay" onClick={handleBackdropClick}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Character Details</h2>
          <button 
            className="modal-close" 
            onClick={onClose}
            aria-label="Close modal"
          >
            ×
          </button>
        </div>

        <div className="modal-content">
          {loading && (
            <div className="loading-item">
              <div className="loading-spinner" />
              Loading character details...
            </div>
          )}

          {error && (
            <div style={{ 
              padding: '1rem', 
              background: '#fef2f2', 
              border: '1px solid #fecaca', 
              borderRadius: '0.5rem',
              color: '#dc2626',
              marginBottom: '1rem'
            }}>
              Failed to load character details: {error}
            </div>
          )}

          {detailedCharacter && (
            <div className="character-detail">
              {/* Large character display */}
              <div className="character-detail-symbol">
                {renderSymbol()}
              </div>

              {/* Character information grid */}
              <div className="character-detail-info">
                <div className="detail-item">
                  <div className="detail-label">Name</div>
                  <div className="detail-value" style={{ fontFamily: 'inherit' }}>
                    {detailedCharacter.name || 'Unknown'}
                  </div>
                </div>

                <div className="detail-item">
                  <div className="detail-label">Codepoint</div>
                  <div className="detail-value">
                    {detailedCharacter.codepoint}
                  </div>
                </div>

                <div className="detail-item">
                  <div className="detail-label">Decimal</div>
                  <div className="detail-value">
                    {detailedCharacter.decimal}
                  </div>
                </div>

                <div className="detail-item">
                  <div className="detail-label">Category</div>
                  <div className="detail-value">
                    {detailedCharacter.category}
                  </div>
                </div>

                <div className="detail-item">
                  <div className="detail-label">Unicode Block</div>
                  <div className="detail-value" style={{ fontFamily: 'inherit' }}>
                    {detailedCharacter.block}
                  </div>
                </div>

                <div className="detail-item">
                  <div className="detail-label">Unicode Version</div>
                  <div className="detail-value">
                    {detailedCharacter.unicode_version}
                  </div>
                </div>
              </div>

              {/* Description */}
              {detailedCharacter.description && (
                <div style={{ 
                  margin: '1.5rem 0', 
                  padding: '1rem', 
                  background: '#f8fafc', 
                  borderRadius: '0.5rem',
                  fontSize: '14px',
                  lineHeight: '1.6'
                }}>
                  <strong>Description:</strong> {detailedCharacter.description}
                </div>
              )}

              {/* Usage examples */}
              <div className="usage-examples">
                <h4>Usage Examples</h4>
                <div className="usage-list">
                  {usageExamples.map((example, index) => (
                    <div 
                      key={index}
                      className="usage-item"
                      onClick={() => copyToClipboard(example.value, example.label)}
                      title={`${example.description} - Click to copy`}
                    >
                      <strong>{example.label}:</strong> {example.value}
                    </div>
                  ))}
                </div>
              </div>

              {/* Additional properties */}
              {detailedCharacter.properties && Object.keys(detailedCharacter.properties).length > 0 && (
                <div style={{ marginTop: '1.5rem' }}>
                  <h4 style={{ fontSize: '14px', marginBottom: '0.75rem' }}>
                    Additional Properties
                  </h4>
                  <div style={{
                    background: '#f8fafc',
                    border: '1px solid #e2e8f0',
                    borderRadius: '0.5rem',
                    padding: '1rem'
                  }}>
                    <pre style={{
                      fontSize: '12px',
                      fontFamily: 'var(--font-mono)',
                      color: '#475569',
                      margin: 0,
                      whiteSpace: 'pre-wrap'
                    }}>
                      {JSON.stringify(detailedCharacter.properties, null, 2)}
                    </pre>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default CharacterModal