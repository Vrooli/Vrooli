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
    <div className={`app-card-skeleton ${viewMode}`}>
      <div className="app-card-header">
        <LoadingSkeleton height="24px" width="60%" />
        <LoadingSkeleton height="16px" width="80px" borderRadius="12px" />
      </div>
      <div className="app-card-content">
        <div className="app-card-metrics">
          <LoadingSkeleton height="14px" width="40%" />
          <LoadingSkeleton height="14px" width="50%" />
          <LoadingSkeleton height="14px" width="35%" />
        </div>
        <div className="app-card-actions">
          <LoadingSkeleton height="32px" width="64px" borderRadius="4px" />
          <LoadingSkeleton height="32px" width="64px" borderRadius="4px" />
          <LoadingSkeleton height="32px" width="80px" borderRadius="4px" />
        </div>
      </div>
    </div>
  );
};

interface ResourceCardSkeletonProps {}

export const ResourceCardSkeleton: React.FC<ResourceCardSkeletonProps> = () => {
  return (
    <div className="resource-card-skeleton">
      <div className="resource-icon">
        <LoadingSkeleton width="40px" height="40px" borderRadius="8px" />
      </div>
      <LoadingSkeleton height="18px" width="70%" />
      <LoadingSkeleton height="14px" width="50%" />
      <LoadingSkeleton height="16px" width="60px" borderRadius="12px" />
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
    <div className={`apps-grid-skeleton ${viewMode}`}>
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
    <div className="resources-grid-skeleton">
      {Array.from({ length: count }, (_, index) => (
        <ResourceCardSkeleton key={index} />
      ))}
    </div>
  );
};

export default LoadingSkeleton;