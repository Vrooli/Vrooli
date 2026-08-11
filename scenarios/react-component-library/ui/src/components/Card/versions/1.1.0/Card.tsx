/**
 * @vrooliComponentSource react-component-library:Card
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 5c10e008-e3b8-4f5e-a99c-c4d3b022ce19
 * @vrooliComponentAppliedAt 2026-08-11T00:08:35Z
 * @vrooliComponentSourceSha256 1f544daab2f4f3efa04cb9f66e689cfbc82f90b3a5efb014a4ed6144cacaa0be
 * @vrooliComponentDriftHash 8b4d17efb05baa35ce7981111aff11e6170fe30e37e1c2d24e8dec94661ad64b
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { HTMLAttributes, ReactNode } from "react";
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

export interface CardDescriptionProps extends HTMLAttributes<HTMLParagraphElement> {
  children: ReactNode;
}

export interface CardContentProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

export function Card({ children, className, ...props }: CardProps) {
  return (
    <div className={cn("rcl-card rounded-panel", className)} data-rcl-card {...props}>
      <style data-rcl-card-styles dangerouslySetInnerHTML={{ __html: cardStyles }} />
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

export function CardDescription({ children, className, ...props }: CardDescriptionProps) {
  return (
    <p className={cn("rcl-card__description", className)} {...props}>
      {children}
    </p>
  );
}

export function CardContent({ children, className, ...props }: CardContentProps) {
  return (
    <div className={cn("rcl-card__content", className)} {...props}>
      {children}
    </div>
  );
}
