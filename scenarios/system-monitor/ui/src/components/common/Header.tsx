import { RefreshCw, Terminal } from 'lucide-react';

interface HeaderProps {
  isOnline: boolean;
  unreadErrorCount: number;
  onRefresh: () => void;
  onToggleTerminal: () => void;
}

export const Header = ({ isOnline, unreadErrorCount, onRefresh, onToggleTerminal }: HeaderProps) => {
  return (
    <header className="header" style={{ 
      background: 'rgba(0, 0, 0, 0.9)',
      borderBottom: '1px solid var(--color-accent)',
      padding: '1rem 0'
    }}>
      <div className="container" style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        maxWidth: '1400px',
        margin: '0 auto',
        padding: '0 2rem'
      }}>
        <h1 className="logo" style={{ 
          margin: 0,
          fontSize: 'var(--font-size-xl)',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          <span className="bracket" style={{ color: 'var(--color-accent)' }}>[</span>
          <span className="text matrix-text-glow">SYSTEM MONITOR</span>
          <span className="bracket" style={{ color: 'var(--color-accent)' }}>]</span>
        </h1>
        
        <div className="status-indicator">
          <span 
            className={`status-dot ${isOnline ? '' : 'offline'}`}
            style={{
              width: '12px',
              height: '12px',
              borderRadius: '50%',
              background: isOnline ? 'var(--color-success)' : 'var(--color-error)',
              boxShadow: isOnline 
                ? '0 0 10px var(--color-success)' 
                : '0 0 10px var(--color-error)',
              animation: 'pulse 2s infinite'
            }}
          />
          
          <span className="status-text" style={{ marginLeft: '0.5rem', marginRight: '1rem' }}>
            {isOnline ? 'ONLINE' : 'OFFLINE'}
          </span>
          
          <button 
            className="btn btn-action"
            onClick={onRefresh}
            title="Refresh Dashboard"
            style={{ marginRight: '0.5rem' }}
          >
            <RefreshCw size={16} />
            REFRESH
          </button>
          
          <button 
            className="btn btn-action"
            onClick={onToggleTerminal}
            title="Toggle System Output"
            style={{ position: 'relative' }}
          >
            <Terminal size={16} />
            OUTPUT
            {unreadErrorCount > 0 && (
              <span 
                className="notification-badge"
                style={{
                  position: 'absolute',
                  top: '-8px',
                  right: '-8px',
                  background: 'var(--color-error)',
                  color: 'var(--color-background)',
                  padding: '2px 6px',
                  borderRadius: '50%',
                  fontSize: 'var(--font-size-xs)',
                  fontWeight: 'bold',
                  minWidth: '18px',
                  textAlign: 'center'
                }}
              >
                {unreadErrorCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
};