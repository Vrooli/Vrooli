import { Settings, Terminal } from 'lucide-react';
import { StatusIndicator } from './StatusIndicator';

interface HeaderProps {
  isOnline: boolean;
  unreadErrorCount: number;
  onToggleTerminal: () => void;
  onOpenSettings: () => void;
}

export const Header = ({ isOnline, unreadErrorCount, onToggleTerminal, onOpenSettings }: HeaderProps) => {
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
        
        <div className="header-controls">
          <StatusIndicator fallbackOnline={isOnline} />

          <button
            className="header-button icon-button"
            onClick={onOpenSettings}
            type="button"
            title="Open system settings"
          >
            <Settings size={16} />
          </button>

          <button
            className="header-button icon-button"
            onClick={onToggleTerminal}
            type="button"
            title="Toggle system output"
            style={{ position: 'relative' }}
          >
            <Terminal size={16} />
            {unreadErrorCount > 0 && (
              <span 
                className="notification-badge"
                style={{
                  position: 'absolute',
                  top: '-8px',
                  right: '-8px'
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
