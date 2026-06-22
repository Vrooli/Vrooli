import { useState, useEffect } from 'react';
import { X } from 'lucide-react';
import { Modal } from './Modal';
import { formatTime } from '../utils/formatters';
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

    return () => { clearInterval(interval); };
  }, []);

  return (
    <Modal isOpen={isVisible} onClose={onClose} ariaLabel="System output" className="modal-terminal">
      <div className="terminal-modal-header">
        <span id="terminal-heading">System Output</span>
        <button
          type="button"
          className="modal-close"
          onClick={onClose}
          aria-label="Close system output"
        >
          <X size={18} />
        </button>
      </div>

      <div className="terminal-modal-body">
        {lines.map((line) => (
          <div
            key={line.id}
            className={`terminal-line ${line.type}`}
          >
            <span className="text-muted">
              [{formatTime(line.timestamp)}]
            </span>{' '}
            {line.message}
          </div>
        ))}
      </div>
    </Modal>
  );
};
