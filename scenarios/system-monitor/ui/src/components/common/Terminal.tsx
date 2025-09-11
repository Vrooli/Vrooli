import { useState, useEffect } from 'react';
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

  const addLine = (message: string, type: TerminalLine['type'] = 'info') => {
    const newLine: TerminalLine = {
      id: Date.now().toString(),
      timestamp: new Date().toISOString(),
      message,
      type
    };
    
    setLines(prev => [...prev.slice(-49), newLine]); // Keep last 50 lines
  };

  // Simulate some system messages
  useEffect(() => {
    const interval = setInterval(() => {
      const messages = [
        '[DEBUG] Metrics polling active',
        '[INFO] API connection healthy',
        '[DEBUG] Memory usage within normal range',
        '[INFO] No alerts detected'
      ];
      
      const randomMessage = messages[Math.floor(Math.random() * messages.length)];
      addLine(randomMessage, 'info');
    }, 10000); // Every 10 seconds

    return () => clearInterval(interval);
  }, []);

  const getLineColor = (type: TerminalLine['type']): string => {
    switch (type) {
      case 'success': return 'var(--color-success)';
      case 'warning': return 'var(--color-warning)';
      case 'error': return 'var(--color-error)';
      default: return 'var(--color-info)';
    }
  };

  return (
    <div 
      className={`terminal ${isVisible ? 'visible' : ''}`}
      style={{
        position: 'fixed',
        bottom: 0,
        right: 0,
        width: '600px',
        height: '400px',
        background: 'rgba(0, 0, 0, 0.95)',
        border: '1px solid var(--color-accent)',
        borderBottom: 'none',
        borderRight: 'none',
        borderRadius: 'var(--border-radius-lg) 0 0 0',
        display: 'flex',
        flexDirection: 'column',
        transform: isVisible ? 'translateY(0)' : 'translateY(100%)',
        transition: 'transform var(--transition-normal)',
        zIndex: 1000
      }}
    >
      <div 
        className="terminal-header"
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: 'var(--spacing-sm) var(--spacing-md)',
          background: 'rgba(0, 255, 0, 0.1)',
          borderBottom: '1px solid var(--color-accent)',
          fontSize: 'var(--font-size-sm)',
          color: 'var(--color-text-bright)'
        }}
      >
        <span>SYSTEM OUTPUT</span>
        <button 
          className="terminal-close"
          onClick={onClose}
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--color-text)',
            fontSize: 'var(--font-size-lg)',
            cursor: 'pointer',
            padding: 0,
            width: '24px',
            height: '24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}
        >
          <X size={16} />
        </button>
      </div>
      
      <div 
        className="terminal-content"
        style={{
          flex: 1,
          padding: 'var(--spacing-md)',
          overflowY: 'auto',
          fontSize: 'var(--font-size-sm)',
          lineHeight: 1.4
        }}
      >
        {lines.map((line) => (
          <div 
            key={line.id}
            className={`terminal-line ${line.type}`}
            style={{
              marginBottom: 'var(--spacing-xs)',
              wordWrap: 'break-word',
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
  );
};