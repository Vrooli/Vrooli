import { useState } from 'react';
import { Bug } from 'lucide-react';

/**
 * Test component to demonstrate Error Boundary functionality
 * This should only be used in development mode
 */
export const ErrorBoundaryTest = () => {
  const [shouldThrow, setShouldThrow] = useState(false);

  // This will throw an error when shouldThrow is true
  if (shouldThrow) {
    throw new Error('Test error thrown by ErrorBoundaryTest component for demonstration purposes');
  }

  // Only show in development mode
  if (process.env.NODE_ENV !== 'development') {
    return null;
  }

  return (
    <div style={{
      position: 'fixed',
      bottom: '20px',
      right: '20px',
      zIndex: 9999,
      background: 'rgba(255, 0, 0, 0.1)',
      border: '1px solid var(--color-error)',
      borderRadius: 'var(--border-radius-md)',
      padding: 'var(--spacing-md)',
      maxWidth: '300px'
    }}>
      <h4 style={{
        margin: '0 0 var(--spacing-sm) 0',
        color: 'var(--color-error)',
        fontSize: 'var(--font-size-sm)',
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--spacing-xs)'
      }}>
        <Bug size={16} />
        Error Boundary Test
      </h4>
      
      <p style={{
        margin: '0 0 var(--spacing-sm) 0',
        color: 'var(--color-text-dim)',
        fontSize: 'var(--font-size-xs)',
        lineHeight: '1.4'
      }}>
        Click to test error boundary functionality (development only)
      </p>
      
      <button
        onClick={() => setShouldThrow(true)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--spacing-xs)',
          padding: 'var(--spacing-xs) var(--spacing-sm)',
          background: 'var(--color-error)',
          color: 'var(--color-background)',
          border: 'none',
          borderRadius: 'var(--border-radius-sm)',
          fontSize: 'var(--font-size-xs)',
          fontWeight: 'bold',
          textTransform: 'uppercase',
          cursor: 'pointer',
          transition: 'all 0.2s'
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.transform = 'scale(1.05)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.transform = 'scale(1)';
        }}
      >
        <Bug size={12} />
        Throw Error
      </button>
    </div>
  );
};