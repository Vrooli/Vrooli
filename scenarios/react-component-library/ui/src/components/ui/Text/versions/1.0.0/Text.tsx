/**
 * @vrooliComponentSource primitives.text
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 62b28b9a-0042-4245-ad96-d8012e4a9d4f
 * @vrooliComponentAppliedAt 2026-08-12T12:59:52Z
 * @vrooliComponentSourceSha256 ce96f4cd7f60483fa03f3a8439524ac75a6d937d0fac6d7e38d98326a66da762
 * @vrooliComponentDriftHash bdcfc4cc77023a807202cf6585dce859eb788209e60048eef2f58f8971a2201b
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { CSSProperties, ElementType, HTMLAttributes, ReactNode } from "react";
import {
  SEMANTIC_TOKENS,
  TEXT_STYLES,
  type TextStyle as TokenTextStyle,
} from "./foundations/Tokens";

export type TextStyle = TokenTextStyle;
export type TextTone = "default" | "muted" | "accent" | "danger";

export interface TextProps extends Omit<HTMLAttributes<HTMLElement>, "style"> {
  /** The bundled typography style. `style="body"` remains supported for compatibility. */
  style?: TextStyle | CSSProperties;
  textStyle?: TextStyle;
  as?: ElementType;
  tone?: TextTone;
  truncate?: boolean;
  balance?: boolean;
  numeric?: boolean;
  children?: ReactNode;
}

const toneColors: Record<TextTone, string> = {
  default: SEMANTIC_TOKENS.foreground,
  muted: SEMANTIC_TOKENS.muted,
  accent: SEMANTIC_TOKENS.primary,
  danger: SEMANTIC_TOKENS.danger,
};

export function Text({
  style,
  textStyle = "body",
  as = "span",
  tone = "default",
  truncate = false,
  balance = false,
  numeric = false,
  children,
  className,
  ...props
}: TextProps) {
  const Component: ElementType = as;
  const selectedStyle: TextStyle = typeof style === "string" ? style : textStyle;
  const inlineStyle = typeof style === "object" ? style : undefined;
  const textCSS: CSSProperties = {
    color: toneColors[tone],
    font: TEXT_STYLES[selectedStyle],
    ...(balance ? { textWrap: "balance" } : {}),
    ...(numeric ? { fontVariantNumeric: "tabular-nums", whiteSpace: "nowrap" } : {}),
    ...(truncate ? { overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" } : {}),
    ...inlineStyle,
  };
  return (
    <Component
      {...props}
      className={className}
      data-text-style={selectedStyle}
      data-text-tone={tone}
      data-text-truncate={truncate || undefined}
      data-text-balance={balance || undefined}
      data-text-numeric={numeric || undefined}
      style={textCSS}
    >
      {children}
    </Component>
  );
}
