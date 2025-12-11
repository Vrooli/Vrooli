import type { SubscriptionTier } from '@stores/entitlementStore';
import { TIER_CONFIG } from '@stores/entitlementStore';
import { Crown, Sparkles, Star, Zap, User, type LucideIcon } from 'lucide-react';

export interface TierBadgeProps {
  tier: SubscriptionTier;
  size?: 'sm' | 'md' | 'lg';
  showIcon?: boolean;
  showLabel?: boolean;
  className?: string;
  onClick?: () => void;
}

const TIER_ICONS: Record<SubscriptionTier, LucideIcon> = {
  free: User,
  solo: Zap,
  pro: Star,
  studio: Sparkles,
  business: Crown,
};

export function TierBadge({
  tier,
  size = 'md',
  showIcon = true,
  showLabel = true,
  className = '',
  onClick,
}: TierBadgeProps) {
  const config = TIER_CONFIG[tier];
  const Icon = TIER_ICONS[tier];

  const sizeClasses = {
    sm: {
      container: 'px-1.5 py-0.5 text-xs gap-1',
      icon: 12,
    },
    md: {
      container: 'px-2 py-1 text-sm gap-1.5',
      icon: 14,
    },
    lg: {
      container: 'px-3 py-1.5 text-base gap-2',
      icon: 18,
    },
  };

  const Component = onClick ? 'button' : 'span';

  return (
    <Component
      type={onClick ? 'button' : undefined}
      onClick={onClick}
      className={`
        inline-flex items-center rounded-full font-medium
        ${config.bgColor} ${config.color} border ${config.borderColor}
        ${sizeClasses[size].container}
        ${onClick ? 'cursor-pointer hover:opacity-80 transition-opacity' : ''}
        ${className}
      `}
    >
      {showIcon && <Icon size={sizeClasses[size].icon} />}
      {showLabel && <span>{config.label}</span>}
    </Component>
  );
}

export default TierBadge;
