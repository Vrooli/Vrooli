interface LoadingSkeletonProps {
  variant?: 'list' | 'card' | 'simple';
  count?: number;
}

export const LoadingSkeleton = ({ variant = 'list', count = 3 }: LoadingSkeletonProps) => {
  const renderListSkeleton = () => (
    <div 
      style={{
        padding: 'var(--spacing-md)',
        borderBottom: '1px solid var(--color-accent)',
        background: 'rgba(0, 0, 0, 0.2)'
      }}
    >
      {/* Header skeleton */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-xs)'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
          <div 
            className="skeleton-item"
            style={{
              width: '16px',
              height: '16px',
              borderRadius: '2px'
            }}
          />
          <div 
            className="skeleton-item"
            style={{
              width: '150px',
              height: '18px',
              borderRadius: '2px'
            }}
          />
        </div>
        <div 
          className="skeleton-item"
          style={{
            width: '70px',
            height: '14px',
            borderRadius: '2px'
          }}
        />
      </div>
      
      {/* Metadata skeleton */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        marginBottom: 'var(--spacing-sm)'
      }}>
        <div 
          className="skeleton-item"
          style={{
            width: '120px',
            height: '14px',
            borderRadius: '2px'
          }}
        />
        <div 
          className="skeleton-item"
          style={{
            width: '80px',
            height: '14px',
            borderRadius: '2px'
          }}
        />
      </div>
      
      {/* Description skeleton */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
        <div 
          className="skeleton-item"
          style={{
            width: '100%',
            height: '14px',
            borderRadius: '2px'
          }}
        />
        <div 
          className="skeleton-item"
          style={{
            width: '80%',
            height: '14px',
            borderRadius: '2px'
          }}
        />
      </div>
    </div>
  );

  const renderCardSkeleton = () => (
    <div 
      style={{
        padding: 'var(--spacing-md)',
        background: 'rgba(0, 0, 0, 0.5)',
        borderRadius: 'var(--border-radius-md)',
        marginBottom: 'var(--spacing-md)'
      }}
    >
      {/* Header skeleton */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
        marginBottom: 'var(--spacing-md)'
      }}>
        <div>
          <div 
            className="skeleton-item"
            style={{
              width: '120px',
              height: '18px',
              borderRadius: '2px',
              marginBottom: 'var(--spacing-xs)'
            }}
          />
          <div 
            className="skeleton-item"
            style={{
              width: '200px',
              height: '14px',
              borderRadius: '2px'
            }}
          />
        </div>
        <div 
          className="skeleton-item"
          style={{
            width: '60px',
            height: '20px',
            borderRadius: 'var(--border-radius-sm)'
          }}
        />
      </div>
      
      {/* Content grid skeleton */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
        gap: 'var(--spacing-md)',
        marginBottom: 'var(--spacing-md)'
      }}>
        {Array.from({ length: 4 }).map((_, idx) => (
          <div key={idx}>
            <div 
              className="skeleton-item"
              style={{
                width: '100px',
                height: '12px',
                borderRadius: '2px',
                marginBottom: 'var(--spacing-xs)'
              }}
            />
            <div 
              className="skeleton-item"
              style={{
                width: '60px',
                height: '16px',
                borderRadius: '2px'
              }}
            />
          </div>
        ))}
      </div>
      
      {/* Recommendations skeleton */}
      <div>
        <div 
          className="skeleton-item"
          style={{
            width: '150px',
            height: '14px',
            borderRadius: '2px',
            marginBottom: 'var(--spacing-sm)'
          }}
        />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
          <div 
            className="skeleton-item"
            style={{
              width: '90%',
              height: '12px',
              borderRadius: '2px'
            }}
          />
          <div 
            className="skeleton-item"
            style={{
              width: '85%',
              height: '12px',
              borderRadius: '2px'
            }}
          />
        </div>
      </div>
    </div>
  );

  const renderSimpleSkeleton = () => (
    <div style={{
      textAlign: 'center',
      padding: 'var(--spacing-lg)',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      gap: 'var(--spacing-md)'
    }}>
      <div 
        className="skeleton-pulse"
        style={{
          width: '32px',
          height: '32px',
          borderRadius: '50%',
          background: 'linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.3) 50%, rgba(0,255,0,0.1) 75%)',
          backgroundSize: '200% 100%',
          animation: 'skeleton-loading 1.5s infinite'
        }}
      />
      <div style={{
        color: 'var(--color-text-dim)',
        fontSize: 'var(--font-size-lg)',
        textTransform: 'uppercase',
        letterSpacing: '2px'
      }}>
        LOADING...
      </div>
    </div>
  );

  return (
    <div className="loading-skeleton">
      <style>{`
        @keyframes skeleton-loading {
          0% { background-position: -200% 0; }
          100% { background-position: 200% 0; }
        }
        
        @keyframes skeleton-pulse {
          0%, 100% { opacity: 0.3; }
          50% { opacity: 0.7; }
        }
        
        .skeleton-item {
          background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.3) 50%, rgba(0,255,0,0.1) 75%);
          background-size: 200% 100%;
          animation: skeleton-loading 1.5s infinite;
          border-radius: 2px;
        }
        
        .skeleton-pulse {
          animation: skeleton-pulse 1.5s infinite;
        }
      `}</style>
      
      {variant === 'simple' ? (
        renderSimpleSkeleton()
      ) : (
        Array.from({ length: count }).map((_, index) => (
          <div key={index}>
            {variant === 'list' ? renderListSkeleton() : renderCardSkeleton()}
          </div>
        ))
      )}
    </div>
  );
};