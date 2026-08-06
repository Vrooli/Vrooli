/** @vrooliComponentSource foundations.visual-recipes */
import { COMPONENT_TOKENS, SEMANTIC_TOKENS } from '../../../Tokens/versions/1.0.0/Tokens';

export const recipe = (...classes: Array<string | false | null | undefined>) => classes.filter(Boolean).join(' ');
export const CONTROL_VARIANTS = {
  primary: recipe('bg-[var(--app-primary)]', 'text-[var(--app-primary-foreground)]', 'border-transparent'),
  secondary: recipe('bg-[var(--app-surface)]', 'text-[var(--app-foreground)]', 'border-[var(--app-border)]'),
  ghost: recipe('bg-transparent', 'text-[var(--app-foreground)]', 'border-transparent'),
  danger: recipe('bg-[var(--app-danger)]', 'text-[var(--app-primary-foreground)]', 'border-transparent'),
} as const;
export const CONTROL_SIZES = {
  sm: recipe('min-h-[var(--control-height-sm)]', 'px-[var(--space-sm)]', 'text-[var(--text-label)]'),
  md: recipe(`min-h-[${COMPONENT_TOKENS.controlHeight}]`, `px-[${COMPONENT_TOKENS.controlPadding}]`, `text-[${SEMANTIC_TOKENS.foreground}]`),
  lg: recipe('min-h-[var(--control-height-lg)]', 'px-[var(--space-md)]', 'text-[var(--text-label)]'),
} as const;
export const SURFACE_ELEVATIONS = {
  flat: 'shadow-[var(--elev-flat)]', raised: 'shadow-[var(--elev-raised)]', floating: 'shadow-[var(--elev-floating)]', overlay: 'shadow-[var(--elev-overlay)]',
} as const;
export type ControlVariant = keyof typeof CONTROL_VARIANTS;
export type ControlSize = keyof typeof CONTROL_SIZES;
