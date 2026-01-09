import React from 'react';
import './LoadingSkeleton.css';

interface LoadingSkeletonProps {
  className?: string;
  width?: string | number;
  height?: string | number;
  borderRadius?: string | number;
}

export const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  className = '',
  width = '100%',
  height = '1em',
  borderRadius = '4px',
}) => {
  return (
    <div
      className={`loading-skeleton ${className}`}
      style={{
        width,
        height,
        borderRadius,
      }}
    />
  );
};

interface AppCardSkeletonProps {
  viewMode?: 'grid' | 'list';
}

export const AppCardSkeleton: React.FC<AppCardSkeletonProps> = ({ viewMode = 'grid' }) => {
  return (
    <div className={`surface-card-skeleton ${viewMode}`}>
      <div className="surface-card-skeleton__thumb" aria-hidden />
      <div className="surface-card-skeleton__body">
        <LoadingSkeleton height="18px" width="70%" />
        <div className="surface-card-skeleton__chips">
          <LoadingSkeleton height="14px" width="42%" borderRadius="999px" />
          <LoadingSkeleton height="14px" width="32%" borderRadius="999px" />
        </div>
      </div>
    </div>
  );
};

interface ResourceCardSkeletonProps {}

export const ResourceCardSkeleton: React.FC<ResourceCardSkeletonProps> = () => {
  return (
    <div className="surface-card-skeleton surface-card-skeleton--resource">
      <div className="surface-card-skeleton__thumb surface-card-skeleton__thumb--resource" aria-hidden />
      <div className="surface-card-skeleton__body">
        <LoadingSkeleton height="18px" width="65%" />
        <div className="surface-card-skeleton__chips">
          <LoadingSkeleton height="14px" width="40%" borderRadius="999px" />
          <LoadingSkeleton height="14px" width="28%" borderRadius="999px" />
        </div>
      </div>
    </div>
  );
};

interface AppsGridSkeletonProps {
  count?: number;
  viewMode?: 'grid' | 'list';
}

export const AppsGridSkeleton: React.FC<AppsGridSkeletonProps> = ({ 
  count = 6, 
  viewMode = 'grid' 
}) => {
  return (
    <div className={`surface-grid-skeleton ${viewMode}`}>
      {Array.from({ length: count }, (_, index) => (
        <AppCardSkeleton key={index} viewMode={viewMode} />
      ))}
    </div>
  );
};

interface ResourcesGridSkeletonProps {
  count?: number;
}

export const ResourcesGridSkeleton: React.FC<ResourcesGridSkeletonProps> = ({ 
  count = 4 
}) => {
  return (
    <div className="surface-grid-skeleton surface-grid-skeleton--compact">
      {Array.from({ length: count }, (_, index) => (
        <ResourceCardSkeleton key={index} />
      ))}
    </div>
  );
};

export default LoadingSkeleton;
