import { useState, useEffect, useRef } from 'react';
import { X } from 'lucide-react';
import type { TerminalLine } from '../../types';

interface TerminalProps {
  isVisible: boolean;
  onClose: () => void;
}

export const Terminal = ({ isVisible, onClose }: TerminalProps) => {
  const [lines, setLines] = useState<TerminalLine[]>([
    {
      id: '1',
      timestamp: new Date().toISOString(),
      message: '[SYSTEM] System Monitor initialized',
      type: 'success'
    },
    {
      id: '2',
      timestamp: new Date().toISOString(),
      message: '[INFO] React/TypeScript UI loaded',
      type: 'info'
    }
  ]);

  const dialogRef = useRef<HTMLDivElement | null>(null);

  // Simulate some system messages
  useEffect(() => {
    const messages = [
      '[DEBUG] Metrics polling active',
      '[INFO] API connection healthy',
      '[DEBUG] Memory usage within normal range',
      '[INFO] No alerts detected'
    ];

    const interval = setInterval(() => {
      const randomIndex = Math.floor(Math.random() * messages.length);
      const randomMessage = messages[randomIndex] ?? messages[0] ?? '[INFO] System active';

      const newLine: TerminalLine = {
        id: Date.now().toString(),
        timestamp: new Date().toISOString(),
        message: randomMessage,
        type: 'info'
      };
      setLines(prev => [...prev.slice(-49), newLine]);
    }, 10000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!isVisible) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isVisible, onClose]);

  useEffect(() => {
    if (isVisible && dialogRef.current) {
      dialogRef.current.focus();
    }
  }, [isVisible]);

  const getLineColor = (type: TerminalLine['type']): string => {
    switch (type) {
      case 'success': return 'var(--color-success)';
      case 'warning': return 'var(--color-warning)';
      case 'error': return 'var(--color-error)';
      default: return 'var(--color-info)';
    }
  };

  if (!isVisible) {
    return null;
  }

  return (
    <div
      className="modal-overlay"
      style={{ cursor: 'auto' }}
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="terminal-heading"
        ref={dialogRef}
        tabIndex={-1}
        onClick={(event) => event.stopPropagation()}
        style={{
          width: 'min(900px, calc(100vw - 3rem))',
          maxHeight: 'min(80vh, 720px)',
          display: 'flex',
          flexDirection: 'column',
          background: 'var(--color-surface)',
          border: '1px solid var(--color-surface-border)',
          borderRadius: 'var(--border-radius-lg)',
          boxShadow: 'var(--shadow-bright-glow)'
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: 'var(--spacing-md) var(--spacing-lg)',
            borderBottom: '1px solid var(--color-surface-border)',
            background: 'var(--alpha-accent-08)',
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            fontSize: 'var(--font-size-sm)'
          }}
        >
          <span id="terminal-heading">System Output</span>
          <button
            type="button"
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--color-text)',
              fontSize: 'var(--font-size-lg)',
              cursor: 'pointer'
            }}
            aria-label="Close system output"
          >
            <X size={18} />
          </button>
        </div>

        <div
          style={{
            flex: 1,
            padding: 'var(--spacing-lg)',
            overflowY: 'auto',
            fontSize: 'var(--font-size-sm)',
            lineHeight: 1.45
          }}
        >
          {lines.map((line) => (
            <div
              key={line.id}
              className={`terminal-line ${line.type}`}
              style={{
                marginBottom: 'var(--spacing-xs)',
                wordBreak: 'break-word',
                color: getLineColor(line.type)
              }}
            >
              <span style={{ color: 'var(--color-text-dim)' }}>
                [{new Date(line.timestamp).toLocaleTimeString()}]
              </span>{' '}
              {line.message}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
