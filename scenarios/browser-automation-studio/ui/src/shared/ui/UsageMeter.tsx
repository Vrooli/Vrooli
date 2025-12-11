import { useMemo } from 'react';

export interface UsageMeterProps {
  used: number;
  limit: number; // -1 for unlimited
  label?: string;
  showText?: boolean;
  size?: 'sm' | 'md' | 'lg';
  variant?: 'bar' | 'circular';
  className?: string;
}

// Get color based on usage percentage
const getColorForPercentage = (percentage: number): string => {
  if (percentage >= 90) return 'bg-red-500';
  if (percentage >= 75) return 'bg-amber-500';
  if (percentage >= 50) return 'bg-yellow-500';
  return 'bg-green-500';
};

const getTextColorForPercentage = (percentage: number): string => {
  if (percentage >= 90) return 'text-red-400';
  if (percentage >= 75) return 'text-amber-400';
  if (percentage >= 50) return 'text-yellow-400';
  return 'text-green-400';
};

export function UsageMeter({
  used,
  limit,
  label,
  showText = true,
  size = 'md',
  variant = 'bar',
  className = '',
}: UsageMeterProps) {
  const isUnlimited = limit === -1;
  const percentage = useMemo(() => {
    if (isUnlimited) return 0;
    if (limit === 0) return 100;
    return Math.min(100, Math.round((used / limit) * 100));
  }, [used, limit, isUnlimited]);

  const sizeClasses = {
    sm: { bar: 'h-1.5', text: 'text-xs', circular: 'w-8 h-8' },
    md: { bar: 'h-2', text: 'text-sm', circular: 'w-12 h-12' },
    lg: { bar: 'h-3', text: 'text-base', circular: 'w-16 h-16' },
  };

  const barColor = getColorForPercentage(percentage);
  const textColor = getTextColorForPercentage(percentage);

  if (variant === 'circular') {
    const strokeWidth = size === 'sm' ? 3 : size === 'lg' ? 5 : 4;
    const radius = size === 'sm' ? 12 : size === 'lg' ? 26 : 18;
    const circumference = 2 * Math.PI * radius;
    const strokeDashoffset = isUnlimited ? circumference : circumference - (percentage / 100) * circumference;

    return (
      <div className={`flex flex-col items-center gap-1 ${className}`}>
        <div className={`relative ${sizeClasses[size].circular}`}>
          <svg className="w-full h-full transform -rotate-90" viewBox="0 0 64 64">
            {/* Background circle */}
            <circle
              cx="32"
              cy="32"
              r={radius}
              fill="none"
              stroke="currentColor"
              strokeWidth={strokeWidth}
              className="text-gray-700"
            />
            {/* Progress circle */}
            {!isUnlimited && (
              <circle
                cx="32"
                cy="32"
                r={radius}
                fill="none"
                stroke="currentColor"
                strokeWidth={strokeWidth}
                strokeLinecap="round"
                strokeDasharray={circumference}
                strokeDashoffset={strokeDashoffset}
                className={barColor.replace('bg-', 'text-')}
                style={{ transition: 'stroke-dashoffset 0.3s ease-in-out' }}
              />
            )}
          </svg>
          {/* Center text */}
          <div className="absolute inset-0 flex items-center justify-center">
            <span className={`${sizeClasses[size].text} font-medium ${isUnlimited ? 'text-gray-400' : textColor}`}>
              {isUnlimited ? '∞' : `${percentage}%`}
            </span>
          </div>
        </div>
        {showText && (
          <span className={`${sizeClasses[size].text} text-gray-400`}>
            {isUnlimited ? 'Unlimited' : `${used}/${limit}`}
          </span>
        )}
        {label && <span className={`${sizeClasses[size].text} text-gray-500`}>{label}</span>}
      </div>
    );
  }

  // Bar variant
  return (
    <div className={`w-full ${className}`}>
      {(label || showText) && (
        <div className="flex items-center justify-between mb-1">
          {label && <span className={`${sizeClasses[size].text} text-gray-400`}>{label}</span>}
          {showText && (
            <span className={`${sizeClasses[size].text} ${isUnlimited ? 'text-gray-400' : textColor}`}>
              {isUnlimited ? 'Unlimited' : `${used} / ${limit}`}
            </span>
          )}
        </div>
      )}
      <div
        className={`w-full bg-gray-700 rounded-full overflow-hidden ${sizeClasses[size].bar}`}
        role="progressbar"
        aria-valuenow={isUnlimited ? undefined : used}
        aria-valuemin={0}
        aria-valuemax={isUnlimited ? undefined : limit}
        aria-label={label || 'Usage'}
      >
        {isUnlimited ? (
          <div
            className="h-full bg-gradient-to-r from-green-500 via-blue-500 to-purple-500 animate-pulse"
            style={{ width: '100%' }}
          />
        ) : (
          <div
            className={`h-full ${barColor} transition-all duration-300 ease-in-out`}
            style={{ width: `${percentage}%` }}
          />
        )}
      </div>
    </div>
  );
}

export default UsageMeter;
