/**
 * @libraryId react-component-library:Card
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 */
import type { CSSProperties, HTMLAttributes, ReactNode } from "react";
import { useSurfaceContext } from "@vrooli/react-component-library/Contracts/1.0.0";
import { cardStyles } from "./styles";
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

export interface CardDescriptionProps
  extends HTMLAttributes<HTMLParagraphElement> {
  children: ReactNode;
}

export interface CardContentProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

const cn = (...inputs: Array<string | undefined>) =>
  inputs.filter(Boolean).join(" ");

export function Card({ children, className, ...props }: CardProps) {
  const { elevation = "flat" } = useSurfaceContext();
  const surfaceStyle: CSSProperties = {
    boxShadow: `var(--elev-${elevation})`,
    ...props.style,
  };
  return (
    <div
      className={cn("rcl-card rounded-panel", className)}
      data-rcl-card
      data-rcl-surface-elevation={elevation}
      {...props}
      style={surfaceStyle}
    >
      <style
        data-rcl-card-styles
        dangerouslySetInnerHTML={{ __html: cardStyles }}
      />
      {children}
    </div>
  );
}

export function CardHeader({ children, className, ...props }: CardHeaderProps) {
  return (
    <div className={cn("rcl-card__header", className)} {...props}>
      {children}
    </div>
  );
}

export function CardTitle({ children, className, ...props }: CardTitleProps) {
  return (
    <h3 className={cn("rcl-card__title", className)} {...props}>
      {children}
    </h3>
  );
}

export function CardDescription({
  children,
  className,
  ...props
}: CardDescriptionProps) {
  return (
    <p className={cn("rcl-card__description", className)} {...props}>
      {children}
    </p>
  );
}

export function CardContent({
  children,
  className,
  ...props
}: CardContentProps) {
  return (
    <div className={cn("rcl-card__content", className)} {...props}>
      {children}
    </div>
  );
}
