/**
 * @libraryId react-component-library:Text
 * @displayName Text
 * @description Text selects one of the eight bundled text styles without free-form font sizing.
 * @version 1.1.1
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.text */
import type { ElementType, HTMLAttributes, ReactNode, Ref } from "react";
import { forwardRef } from "react";
import { cn } from "@vrooli/react-component-library/ClassMerge/1.0.1";
import type { TextStyle as TokenTextStyle } from "@vrooli/react-component-library/Tokens/1.0.0";
import "./Text.css";

export type TextStyle = TokenTextStyle;
export type TextTone = "default" | "muted" | "accent" | "danger";

export interface TextProps extends Omit<HTMLAttributes<HTMLElement>, "style"> {
  textStyle?: TextStyle;
  as?: ElementType;
  tone?: TextTone;
  truncate?: boolean;
  balance?: boolean;
  numeric?: boolean;
  children?: ReactNode;
}

export const Text = forwardRef<HTMLElement, TextProps>(function Text(
  {
    textStyle = "body",
    as = "span",
    tone = "default",
    truncate = false,
    balance = false,
    numeric = false,
    children,
    className,
    ...props
  }: TextProps,
  ref: Ref<HTMLElement>,
) {
  const Component: ElementType = as;
  const selectedStyle: TextStyle = textStyle;
  return (
    <Component
      data-testid="primitives.text"
      {...props}
      ref={ref}
      className={cn("rcl-text", className)}
      data-text-style={selectedStyle}
      data-text-tone={tone}
      data-text-truncate={truncate || undefined}
      data-text-balance={balance || undefined}
      data-text-numeric={numeric || undefined}
    >
      {children}
    </Component>
  );
});
