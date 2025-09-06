import { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { terminalService } from '@/services/api';
import './TerminalView.css';

export default function TerminalView() {
  const [output, setOutput] = useState<string[]>([
    'VROOLI TERMINAL v3.0 - INITIALIZED',
    'Type "help" for available commands',
    ''
  ]);
  const [command, setCommand] = useState('');
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [processing, setProcessing] = useState(false);
  const outputEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [output]);

  const scrollToBottom = () => {
    outputEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleCommand = async (cmd: string) => {
    if (!cmd.trim()) return;

    // Add command to output
    setOutput(prev => [...prev, `vrooli@matrix:~$ ${cmd}`]);
    
    // Add to history
    setHistory(prev => [...prev, cmd]);
    setHistoryIndex(-1);
    
    // Clear input
    setCommand('');
    setProcessing(true);

    try {
      // Handle built-in commands
      if (cmd === 'clear') {
        setOutput([]);
      } else if (cmd === 'help') {
        setOutput(prev => [...prev,
          '',
          'Available commands:',
          '  help     - Show this help message',
          '  clear    - Clear terminal output',
          '  status   - Show system status',
          '  list     - List all applications',
          '  restart  - Restart an application',
          '  stop     - Stop an application',
          '  start    - Start an application',
          ''
        ]);
      } else {
        // Execute command via API
        const result = await terminalService.executeCommand(cmd);
        const lines = result.split('\n');
        setOutput(prev => [...prev, ...lines, '']);
      }
    } catch (error) {
      setOutput(prev => [...prev, 
        `Error: ${error instanceof Error ? error.message : 'Command execution failed'}`,
        ''
      ]);
    } finally {
      setProcessing(false);
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleCommand(command);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (history.length > 0 && historyIndex < history.length - 1) {
        const newIndex = historyIndex === -1 ? 0 : historyIndex + 1;
        setHistoryIndex(newIndex);
        setCommand(history[history.length - 1 - newIndex]);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndex > 0) {
        const newIndex = historyIndex - 1;
        setHistoryIndex(newIndex);
        setCommand(history[history.length - 1 - newIndex]);
      } else if (historyIndex === 0) {
        setHistoryIndex(-1);
        setCommand('');
      }
    }
  };

  return (
    <div className="terminal-view">
      <div className="panel-header">
        <h2>COMMAND TERMINAL</h2>
        <div className="panel-controls">
          <button 
            className="control-btn" 
            onClick={() => setOutput([])}
          >
            CLEAR
          </button>
        </div>
      </div>
      
      <div className="terminal-container" onClick={() => inputRef.current?.focus()}>
        <div className="terminal-output">
          {output.map((line, index) => (
            <div 
              key={index} 
              className={`terminal-line ${line.startsWith('Error:') ? 'error' : ''}`}
            >
              {line}
            </div>
          ))}
          {processing && (
            <div className="terminal-line processing">
              Processing...
            </div>
          )}
          <div ref={outputEndRef} />
        </div>
        <div className="terminal-input-line">
          <span className="terminal-prompt">vrooli@matrix:~$</span>
          <input
            ref={inputRef}
            type="text"
            className="terminal-input"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={processing}
            autoFocus
          />
        </div>
      </div>
    </div>
  );
}