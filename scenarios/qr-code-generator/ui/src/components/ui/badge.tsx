import { cn } from '@/lib/utils';

interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  tone?: 'idle' | 'processing' | 'success' | 'error';
}

const toneClassMap: Record<NonNullable<BadgeProps['tone']>, string> = {
  idle: 'border-panelBorder/40 bg-black/45 text-foreground/70',
  processing: 'border-accent/50 bg-accent/10 text-accent animate-pulseSoft',
  success: 'border-accent/60 bg-accent/15 text-accent',
  error: 'border-danger/60 bg-danger/15 text-[#ffbcd2]'
};

export function Badge({ className, tone = 'idle', ...rest }: BadgeProps) {
  return (
    <div className={cn('status-indicator flex items-center gap-2', toneClassMap[tone], className)} {...rest} />
  );
}
