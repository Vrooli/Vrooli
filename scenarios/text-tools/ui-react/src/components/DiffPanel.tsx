import { useState, useRef, ChangeEvent, FC } from 'react'
import '../styles/DiffPanel.css'

type DiffType = 'line' | 'word' | 'character' | 'semantic'

interface DiffOptions {
  type: DiffType
  ignoreWhitespace: boolean
  ignoreCase: boolean
}

interface DiffChange {
  type: 'add' | 'remove' | 'modify'
  lineStart: number
  lineEnd: number
  content: string
}

interface DiffResult {
  changes: DiffChange[]
  similarity_score: number
  summary: string
}

interface DiffStats {
  added: number
  removed: number
  modified: number
}

const DiffPanel: FC = () => {
  const [text1, setText1] = useState('')
  const [text2, setText2] = useState('')
  const [options, setOptions] = useState<DiffOptions>({
    type: 'line',
    ignoreWhitespace: false,
    ignoreCase: false
  })
  const [results, setResults] = useState<DiffResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const file1Ref = useRef<HTMLInputElement>(null)
  const file2Ref = useRef<HTMLInputElement>(null)

  const handleFileLoad = (fileInput: ChangeEvent<HTMLInputElement>, setText: (text: string) => void) => {
    const file = fileInput.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = (e) => {
        const content = e.target?.result as string
        setText(content)
      }
      reader.readAsText(file)
    }
  }

  const handleCompare = async () => {
    if (!text1.trim() || !text2.trim()) {
      setError('Please enter text in both fields')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch('/api/v1/text/diff', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          text1: text1,
          text2: text2,
          options: options
        })
      })

      if (!response.ok) {
        throw new Error(`API Error: ${response.status}`)
      }

      const result: DiffResult = await response.json()
      setResults(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to compare texts')
      console.error('Diff comparison error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  const calculateStats = (changes: DiffChange[]): DiffStats => {
    return changes.reduce(
      (stats, change) => {
        switch (change.type) {
          case 'add':
            stats.added++
            break
          case 'remove':
            stats.removed++
            break
          case 'modify':
            stats.modified++
            break
        }
        return stats
      },
      { added: 0, removed: 0, modified: 0 }
    )
  }

  const renderDiffResults = () => {
    if (!results) return null

    const stats = calculateStats(results.changes)

    return (
      <>
        <div className="diff-stats">
          <div className="diff-stat">
            <span className="stat-added">+{stats.added}</span>
            <span>Added</span>
          </div>
          <div className="diff-stat">
            <span className="stat-removed">-{stats.removed}</span>
            <span>Removed</span>
          </div>
          <div className="diff-stat">
            <span className="stat-modified">~{stats.modified}</span>
            <span>Modified</span>
          </div>
        </div>

        {results.changes.map((change, index) => (
          <div key={index} className={`diff-line ${change.type === 'add' ? 'added' : change.type === 'remove' ? 'removed' : 'modified'}`}>
            <span className="line-number">{change.lineStart}</span>
            <span className="line-content">{change.content}</span>
          </div>
        ))}

        {results.summary && (
          <div style={{ marginTop: '1rem', padding: '0.5rem', backgroundColor: 'var(--bg-tertiary)', borderRadius: '4px' }}>
            <strong>Summary:</strong> {results.summary}
          </div>
        )}
      </>
    )
  }

  return (
    <div className="tool-panel">
      <div className="tool-header">
        <h2>Text Comparison</h2>
        <div className="tool-options">
          <select
            className="form-select"
            value={options.type}
            onChange={(e) => setOptions({ ...options, type: e.target.value as DiffType })}
          >
            <option value="line">Line by Line</option>
            <option value="word">Word by Word</option>
            <option value="character">Character</option>
            <option value="semantic">Semantic</option>
          </select>
          
          <label className="checkbox">
            <input
              type="checkbox"
              checked={options.ignoreWhitespace}
              onChange={(e) => setOptions({ ...options, ignoreWhitespace: e.target.checked })}
            />
            <span>Ignore Whitespace</span>
          </label>
          
          <label className="checkbox">
            <input
              type="checkbox"
              checked={options.ignoreCase}
              onChange={(e) => setOptions({ ...options, ignoreCase: e.target.checked })}
            />
            <span>Ignore Case</span>
          </label>
        </div>
      </div>

      <div className="diff-container">
        <div className="diff-pane">
          <div className="pane-header">
            <span>Original Text</span>
            <div className="pane-actions">
              <button className="btn-small" onClick={() => setText1('')}>
                Clear
              </button>
              <input
                ref={file1Ref}
                type="file"
                accept=".txt,.md,.json,.xml,.html"
                onChange={(e) => handleFileLoad(e, setText1)}
                className="hidden-file-input"
              />
              <button className="btn-small" onClick={() => file1Ref.current?.click()}>
                Load File
              </button>
            </div>
          </div>
          <textarea
            className="code-editor"
            value={text1}
            onChange={(e) => setText1(e.target.value)}
            placeholder="Enter or paste original text here..."
          />
        </div>

        <div className="diff-pane">
          <div className="pane-header">
            <span>Modified Text</span>
            <div className="pane-actions">
              <button className="btn-small" onClick={() => setText2('')}>
                Clear
              </button>
              <input
                ref={file2Ref}
                type="file"
                accept=".txt,.md,.json,.xml,.html"
                onChange={(e) => handleFileLoad(e, setText2)}
                className="hidden-file-input"
              />
              <button className="btn-small" onClick={() => file2Ref.current?.click()}>
                Load File
              </button>
            </div>
          </div>
          <textarea
            className="code-editor"
            value={text2}
            onChange={(e) => setText2(e.target.value)}
            placeholder="Enter or paste modified text here..."
          />
        </div>
      </div>

      <div className="action-bar">
        <button
          className="btn-primary"
          onClick={handleCompare}
          disabled={isLoading || !text1.trim() || !text2.trim()}
        >
          {isLoading && <div className="loading-spinner" />}
          Compare
        </button>
        
        {results && (
          <span className="similarity-score">
            Similarity: {(results.similarity_score * 100).toFixed(1)}%
          </span>
        )}
      </div>

      <div className="results-container">
        {error && (
          <div style={{ color: 'var(--danger-color)', marginBottom: '1rem' }}>
            Error: {error}
          </div>
        )}
        
        {!results && !error && (
          <div className="results-empty">
            Select text in both panels and click "Compare" to see differences
          </div>
        )}
        
        {renderDiffResults()}
      </div>
    </div>
  )
}

export default DiffPanel