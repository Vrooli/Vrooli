/**
 * @libraryId react-component-library:Card
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
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

export interface CardDescriptionProps
  extends HTMLAttributes<HTMLParagraphElement> {
  children: ReactNode;
}

export interface CardContentProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

export function Card({ children, className, ...props }: CardProps) {
  return (
    <div
      className={joinClasses(
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
      className={joinClasses(
        "flex min-w-0 flex-col gap-1 border-b border-app-border px-4 py-3",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export function CardTitle({ children, className, ...props }: CardTitleProps) {
  return (
    <h3
      className={joinClasses(
        "text-base font-semibold leading-tight text-app-foreground",
        className,
      )}
      {...props}
    >
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
    <p
      className={joinClasses("text-sm text-app-muted-foreground", className)}
      {...props}
    >
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
    <div className={joinClasses("min-w-0 px-4 py-4", className)} {...props}>
      {children}
    </div>
  );
}
