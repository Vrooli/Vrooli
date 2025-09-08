import React, { memo, useCallback } from 'react'

/**
 * Individual character item component for virtualized list
 * Optimized for performance with React.memo
 */
const CharacterItem = memo(function CharacterItem({ 
  character, 
  onClick, 
  viewMode = 'grid' 
}) {
  const handleClick = useCallback(() => {
    onClick?.(character)
  }, [onClick, character])

  const handleKeyPress = useCallback((e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onClick?.(character)
    }
  }, [onClick, character])

  // Render symbol safely - some characters might not display properly
  const renderSymbol = useCallback(() => {
    try {
      // Handle potentially problematic characters
      if (character.decimal < 32 && character.decimal !== 10 && character.decimal !== 13) {
        return '□' // Show box for control characters
      }
      
      return String.fromCodePoint(character.decimal)
    } catch (error) {
      return '□' // Fallback for invalid codepoints
    }
  }, [character.decimal])

  // Format character name for display
  const displayName = character.name || 'Unknown Character'
  const truncatedName = displayName.length > 50 
    ? `${displayName.substring(0, 50)}...` 
    : displayName

  return (
    <div 
      className="character-item"
      onClick={handleClick}
      onKeyPress={handleKeyPress}
      tabIndex={0}
      role="button"
      aria-label={`Character ${character.name} (${character.codepoint})`}
    >
      {/* Character Symbol */}
      <div className="character-symbol" title={`U+${character.decimal.toString(16).toUpperCase()}`}>
        {renderSymbol()}
      </div>

      {/* Codepoint */}
      <div className="character-codepoint">
        {character.codepoint}
      </div>

      {/* Character Name */}
      <div className="character-name" title={displayName}>
        {truncatedName}
      </div>

      {/* Category Badge */}
      <div className="character-category" title={getCategoryDescription(character.category)}>
        {character.category}
      </div>

      {/* Unicode Block (hidden on mobile) */}
      {viewMode === 'grid' && (
        <div className="character-block" title={character.block}>
          {character.block}
        </div>
      )}
    </div>
  )
})

/**
 * Get human-readable description for Unicode category codes
 */
function getCategoryDescription(code) {
  const categoryMap = {
    'Lu': 'Letter, Uppercase',
    'Ll': 'Letter, Lowercase', 
    'Lt': 'Letter, Titlecase',
    'Lm': 'Letter, Modifier',
    'Lo': 'Letter, Other',
    'Mn': 'Mark, Nonspacing',
    'Mc': 'Mark, Spacing Combining',
    'Me': 'Mark, Enclosing',
    'Nd': 'Number, Decimal Digit',
    'Nl': 'Number, Letter',
    'No': 'Number, Other',
    'Pc': 'Punctuation, Connector',
    'Pd': 'Punctuation, Dash',
    'Ps': 'Punctuation, Open',
    'Pe': 'Punctuation, Close',
    'Pi': 'Punctuation, Initial quote',
    'Pf': 'Punctuation, Final quote',
    'Po': 'Punctuation, Other',
    'Sm': 'Symbol, Math',
    'Sc': 'Symbol, Currency',
    'Sk': 'Symbol, Modifier',
    'So': 'Symbol, Other',
    'Zs': 'Separator, Space',
    'Zl': 'Separator, Line',
    'Zp': 'Separator, Paragraph',
    'Cc': 'Other, Control',
    'Cf': 'Other, Format',
    'Cs': 'Other, Surrogate',
    'Co': 'Other, Private Use',
    'Cn': 'Other, Not Assigned'
  }
  
  return categoryMap[code] || `Category: ${code}`
}

export default CharacterItem