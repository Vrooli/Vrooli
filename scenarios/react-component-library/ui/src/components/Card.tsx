/**
 * @vrooliComponentSource react-component-library:Card
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption ad9dc50c-e8fa-4629-8050-67c6adff2d40
 * @vrooliComponentAppliedAt 2026-08-06T03:45:30Z
 * @vrooliComponentSourceSha256 24f9d0cd36afe648f3cd3432a42fe1dc785f3cd063a957d448bf280d731261b6
 * @vrooliComponentDriftHash 4209a0546c66c0dff60c7eaa018e5efef80efc4f746ecf999252f5de2aa50399
 * @vrooliComponentTokenTranslation bg-app-surface->bg-app-surface,border-app-border->border-app-border,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { HTMLAttributes, ReactNode } from "react";
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

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export function Card({ children, className, ...props }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-panel border border-app-border bg-app-surface text-app-foreground",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export function CardHeader({ children, className, ...props }: CardHeaderProps) {
  return (
    <div
      className={cn("flex min-w-0 flex-col gap-1 border-b border-app-border px-4 py-3", className)}
      {...props}
    >
      {children}
    </div>
  );
}

export function CardTitle({ children, className, ...props }: CardTitleProps) {
  return (
    <h3
      className={cn("text-base font-semibold leading-tight text-app-foreground", className)}
      {...props}
    >
      {children}
    </h3>
  );
}

export function CardDescription({ children, className, ...props }: CardDescriptionProps) {
  return (
    <p className={cn("text-sm text-app-muted-foreground", className)} {...props}>
      {children}
    </p>
  );
}

export function CardContent({ children, className, ...props }: CardContentProps) {
  return (
    <div className={cn("min-w-0 px-4 py-4", className)} {...props}>
      {children}
    </div>
  );
}
