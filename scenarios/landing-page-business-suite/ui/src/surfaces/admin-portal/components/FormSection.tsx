import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { LAYOUT } from '../config/layout.constants';

export interface FormSectionProps {
  /** Section title displayed in the card header */
  title: string;
  /** Optional description shown below the title */
  description?: string;
  /** Lucide icon component for the section */
  icon: LucideIcon;
  /** Tailwind text color class for the icon (e.g., 'text-blue-300') */
  iconColorClass: string;
  /** Content to render inside the card */
  children: ReactNode;
  /** Optional actions (buttons) rendered in the header */
  actions?: ReactNode;
  /** Test ID for the section */
  testId?: string;
  /** Additional className for the Card */
  className?: string;
}

/**
 * FormSection - A standardized card wrapper for admin form sections.
 *
 * Replaces the repeated Card + CardHeader + CardTitle with icon + CardDescription pattern.
 *
 * @example
 * ```tsx
 * <FormSection
 *   title="Site Identity"
 *   description="Your site name, tagline, and brand imagery"
 *   icon={Type}
 *   iconColorClass="text-blue-300"
 * >
 *   <div className="grid gap-6 md:grid-cols-2">
 *     <FormField label="Site Name">
 *       <input type="text" value={form.site_name} onChange={...} />
 *     </FormField>
 *   </div>
 * </FormSection>
 * ```
 */
export function FormSection({
  title,
  description,
  icon: Icon,
  iconColorClass,
  children,
  actions,
  testId,
  className,
}: FormSectionProps) {
  return (
    <Card className={`${LAYOUT.card.base} ${className ?? ''}`} data-testid={testId}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Icon className={`h-5 w-5 ${iconColorClass}`} />
            {title}
          </CardTitle>
          {actions}
        </div>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className={LAYOUT.contentSpacing}>{children}</CardContent>
    </Card>
  );
}
