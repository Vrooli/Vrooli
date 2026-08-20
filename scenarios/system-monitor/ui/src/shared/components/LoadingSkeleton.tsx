interface LoadingSkeletonProps {
  variant?: 'list' | 'card' | 'simple';
  count?: number;
}

export const LoadingSkeleton = ({ variant = 'list', count = 3 }: LoadingSkeletonProps) => {
  const renderListSkeleton = () => (
    <div 
      data-sm-style="sm-style-17abb19013"
    >
      {/* Header skeleton */}
      <div data-sm-style="sm-style-dabe935aa7">
        <div data-sm-style="sm-style-67bc078b69">
          <div 
            className="skeleton-item"
            data-sm-style="sm-style-3dcc0f079d"
          />
          <div 
            className="skeleton-item"
            data-sm-style="sm-style-467d79f189"
          />
        </div>
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-c733575e50"
        />
      </div>
      
      {/* Metadata skeleton */}
      <div data-sm-style="sm-style-48be228cf7">
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-fed0da754e"
        />
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-16bd637909"
        />
      </div>
      
      {/* Description skeleton */}
      <div data-sm-style="sm-style-f42c3fdadb">
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-647322c717"
        />
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-d13567a769"
        />
      </div>
    </div>
  );

  const renderCardSkeleton = () => (
    <div 
      data-sm-style="sm-style-6147d91f78"
    >
      {/* Header skeleton */}
      <div data-sm-style="sm-style-ed56df885d">
        <div>
          <div 
            className="skeleton-item"
            data-sm-style="sm-style-50b259a35d"
          />
          <div 
            className="skeleton-item"
            data-sm-style="sm-style-65cc2b8cef"
          />
        </div>
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-d93100fc92"
        />
      </div>
      
      {/* Content grid skeleton */}
      <div data-sm-style="sm-style-54f14ccf5b">
        {Array.from({ length: 4 }).map((_, idx) => (
          <div key={idx}>
            <div 
              className="skeleton-item"
              data-sm-style="sm-style-73e32f9187"
            />
            <div 
              className="skeleton-item"
              data-sm-style="sm-style-d035611a33"
            />
          </div>
        ))}
      </div>
      
      {/* Recommendations skeleton */}
      <div>
        <div 
          className="skeleton-item"
          data-sm-style="sm-style-7f25ef5fd3"
        />
        <div data-sm-style="sm-style-f42c3fdadb">
          <div 
            className="skeleton-item"
            data-sm-style="sm-style-e0db026113"
          />
          <div 
            className="skeleton-item"
            data-sm-style="sm-style-309db88982"
          />
        </div>
      </div>
    </div>
  );

  const renderSimpleSkeleton = () => (
    <div data-sm-style="sm-style-3ef47af557">
      <div 
        className="skeleton-pulse"
        data-sm-style="sm-style-04bed5fb37"
      />
      <div data-sm-style="sm-style-6ce841334e">
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
          background: linear-gradient(90deg, var(--color-primary-muted) 25%, var(--color-primary-muted) 50%, var(--color-primary-muted) 75%);
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