/**
 * @libraryId react-component-library:Card
 * @displayName Card
 * @version 1.2.4
 * @tags ["surface","layout"]
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { CSSProperties, HTMLAttributes, ReactNode } from "react";
import { useSurfaceContext } from "@vrooli/react-component-library/Contracts/1";
export const cardStyles = `
[data-rcl-card] { min-inline-size: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); }
[data-rcl-card] .rcl-card__header { display: flex; min-inline-size: 0; flex-direction: column; gap: var(--space-2xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-card] .rcl-card__title { color: var(--color-foreground); font-family: var(--font-sans); font-size: var(--text-heading-size); font-weight: 650; line-height: var(--text-heading-line); }
[data-rcl-card] .rcl-card__description { color: var(--color-muted-foreground); font-family: var(--font-sans); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-card] .rcl-card__content { min-inline-size: 0; padding: var(--space-md); }
`;
export const CARD_PARTS = ["header", "media", "body", "footer"] as const;

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export interface CardHeaderProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export interface CardTitleProps extends HTMLAttributes<HTMLHeadingElement> {
  children: ReactNode;
}

export interface CardDescriptionProps extends HTMLAttributes<HTMLParagraphElement> {
  children: ReactNode;
}

export interface CardContentProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

export const Card = withClassName(function Card({ children, className, ...props }: CardProps) {
  const { elevation = "flat" } = useSurfaceContext();
  const testId = (props as CardProps & { "data-testid"?: string })["data-testid"];
  const surfaceStyle: CSSProperties = {
    boxShadow: {
      flat: "var(--elev-flat)",
      raised: "var(--elev-raised)",
      floating: "var(--elev-floating)",
      overlay: "var(--elev-overlay)",
    }[elevation],
    ...props.style,
  };
  return (
    <div
      className={cn("rcl-card", className)}
      data-rcl-card
      data-rcl-surface-elevation={elevation}
      {...props}
      data-testid={testId ?? "primitives.card"}
      style={surfaceStyle}
    >
      <StyleSheet name="card-1-2-1" css={cardStyles} />
      {children}
    </div>
  );
});

export const CardHeader = withClassName(function CardHeader({
  children,
  className,
  ...props
}: CardHeaderProps) {
  return (
    <div className={cn("rcl-card__header", className)} {...props}>
      {children}
    </div>
  );
});

export const CardTitle = withClassName(function CardTitle({
  children,
  className,
  ...props
}: CardTitleProps) {
  return (
    <h3 className={cn("rcl-card__title", className)} {...props}>
      {children}
    </h3>
  );
});

export const CardDescription = withClassName(function CardDescription({
  children,
  className,
  ...props
}: CardDescriptionProps) {
  return (
    <p className={cn("rcl-card__description", className)} {...props}>
      {children}
    </p>
  );
});

export const CardContent = withClassName(function CardContent({
  children,
  className,
  ...props
}: CardContentProps) {
  return (
    <div className={cn("rcl-card__content", className)} {...props}>
      {children}
    </div>
  );
});
