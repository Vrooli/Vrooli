/**
 * @vrooliComponentSource react-component-library:Card
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption d8299794-8721-441e-8916-eb01db2a2fa7
 * @vrooliComponentAppliedAt 2026-07-09T04:31:16Z
 * @vrooliComponentSourceSha256 e03956b8b46e9dd89467b53cdefd9be966f1d1783f5c9ace98b1657e1b3caa34
 * @vrooliComponentDriftHash e03956b8b46e9dd89467b53cdefd9be966f1d1783f5c9ace98b1657e1b3caa34
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { HTMLAttributes, ReactNode } from "react";

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
      className={cn("rounded-panel border border-app-border bg-app-surface text-app-foreground", className)}
      {...props}
    >
      {children}
    </div>
  );
}

export function CardHeader({ children, className, ...props }: CardHeaderProps) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1 border-b border-app-border px-4 py-3", className)} {...props}>
      {children}
    </div>
  );
}

export function CardTitle({ children, className, ...props }: CardTitleProps) {
  return (
    <h3 className={cn("text-base font-semibold leading-tight text-app-foreground", className)} {...props}>
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
