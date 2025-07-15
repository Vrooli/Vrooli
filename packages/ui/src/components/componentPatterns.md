# Tailwind Component Patterns

This document establishes patterns and conventions for creating consistent, performant, and maintainable Tailwind CSS components in the Vrooli UI package.

## Overview

Based on the Button component implementation, this guide provides templates and patterns for building high-quality Tailwind components that follow consistent conventions.

## Component Structure Template

### 1. File Organization

```
/components/
  /componentName/
    ComponentName.tsx        # Main component file
    ComponentName.stories.tsx # Storybook stories
    componentNameStyles.ts   # Style utilities and configuration
    index.ts                # Export file
```

### 2. Component File Template

```tsx
import type { ComponentHTMLAttributes, ReactNode } from "react";
import { forwardRef, useCallback, useMemo } from "react";
import { cn } from "../../utils/tailwind-theme.js";
import { 
    COMPONENT_CONFIG,
    COMPONENT_COLORS,
    buildComponentClasses,
    // ... other utilities
} from "./componentNameStyles.js";

// Export types for external use
export type ComponentVariant = "primary" | "secondary" | "custom";
export type ComponentSize = "sm" | "md" | "lg";

export interface ComponentProps extends ComponentHTMLAttributes<HTMLElement> {
    /** Visual style variant */
    variant?: ComponentVariant;
    /** Size of the component */
    size?: ComponentSize;
    /** Component content */
    children?: ReactNode;
    /** Custom className for overrides */
    className?: string;
}

/**
 * Component description with features and usage examples
 * 
 * @example
 * ```tsx
 * <Component variant="primary" size="md">
 *   Content
 * </Component>
 * ```
 */
export const Component = forwardRef<HTMLElement, ComponentProps>(
    (
        {
            variant = "primary",
            size = "md",
            className,
            children,
            ...props
        },
        ref
    ) => {
        // Memoize classes for performance
        const componentClasses = useMemo(() => 
            buildComponentClasses({
                variant,
                size,
                className,
            }),
            [variant, size, className]
        );

        return (
            <div
                ref={ref}
                className={componentClasses}
                {...props}
            >
                {children}
            </div>
        );
    }
);

Component.displayName = "Component";

// Factory pattern for common variants
export const ComponentFactory = {
    Primary: (props: Omit<ComponentProps, "variant">) => (
        <Component variant="primary" {...props} />
    ),
    Secondary: (props: Omit<ComponentProps, "variant">) => (
        <Component variant="secondary" {...props} />
    ),
} as const;
```

## Styling Patterns

### 1. CSS Class Naming Convention

**Format:** `tw-component-variant-modifier`

**Examples:**
```css
.tw-button-primary          /* Base variant class */
.tw-button-space-container  /* Variant with modifier */
.tw-button-ripple          /* Component effect */
.tw-icon-button-solid      /* Sub-component variant */
```

**Rules:**
- Always prefix with `tw-` to distinguish from utility classes
- Use lowercase and hyphens
- Be specific but concise
- Group related classes together in CSS

### 2. Style Utilities File Pattern

```typescript
// componentNameStyles.ts
import { cn } from "../../utils/tailwind-theme.js";
import type { ComponentVariant, ComponentSize } from "./ComponentName.js";

// Configuration object for constants
export const COMPONENT_CONFIG = {
    // Animation durations, sizes, etc.
    ANIMATION: {
        DURATION: 0.3,
        EASING: "ease-in-out",
    },
    SIZES: {
        sm: 24,
        md: 32,
        lg: 40,
    },
} as const;

// Color system
export const COMPONENT_COLORS = {
    VARIANTS: {
        primary: "#primary-color",
        secondary: "#secondary-color",
    },
} as const;

// Variant styles mapping
export const VARIANT_STYLES: Record<ComponentVariant, string> = {
    primary: cn(
        "tw-bg-primary tw-text-white",
        "hover:tw-bg-primary-dark",
        "focus:tw-ring-2 focus:tw-ring-primary"
    ),
    secondary: cn(
        "tw-bg-secondary tw-text-gray-800",
        "hover:tw-bg-secondary-dark"
    ),
};

// Size styles mapping
export const SIZE_STYLES: Record<ComponentSize, string> = {
    sm: "tw-h-8 tw-px-3 tw-text-xs",
    md: "tw-h-10 tw-px-4 tw-text-sm",
    lg: "tw-h-12 tw-px-6 tw-text-base",
};

// Base styles
export const BASE_COMPONENT_STYLES = cn(
    "tw-inline-flex tw-items-center tw-justify-center",
    "tw-rounded tw-transition-all tw-duration-200",
    "tw-border-0 tw-outline-none"
);

// Utility function to build complete classes
export const buildComponentClasses = ({
    variant = "primary",
    size = "md",
    className,
}: {
    variant?: ComponentVariant;
    size?: ComponentSize;
    className?: string;
}) => {
    return cn(
        BASE_COMPONENT_STYLES,
        VARIANT_STYLES[variant],
        SIZE_STYLES[size],
        className
    );
};
```

### 3. CSS Organization

```css
/* In tailwind.css */
@layer components {
  /* Component base styles */
  .tw-component-base {
    @apply tw-inline-flex tw-items-center tw-justify-center;
    @apply tw-rounded tw-transition-all tw-duration-200;
  }

  /* Variant styles */
  .tw-component-primary {
    @apply tw-component-base;
    @apply tw-bg-primary tw-text-white;
  }

  /* Interactive states */
  .tw-component-primary:hover:not(:disabled) {
    @apply tw-bg-primary-dark;
  }

  /* Special effects */
  .tw-component-ripple {
    @apply tw-absolute tw-pointer-events-none tw-rounded-full;
    animation: componentRipple 0.6s ease-out forwards;
  }
}

/* Animations */
@keyframes componentRipple {
  0% { transform: scale(0); opacity: 1; }
  100% { transform: scale(4); opacity: 0; }
}
```

## Performance Patterns

### 1. Memoization Strategy

```tsx
// Memoize expensive calculations
const componentClasses = useMemo(() => 
    buildComponentClasses({ variant, size, className }),
    [variant, size, className]  // Only dependencies that affect output
);

// Memoize complex derived values
const processedData = useMemo(() => 
    processComplexData(data),
    [data]
);

// Use useCallback for event handlers
const handleClick = useCallback((event) => {
    // Handle event
    onClick?.(event);
}, [onClick]);
```

### 2. CSS-First Approach

- Move styles to CSS classes instead of inline styles when possible
- Use CSS custom properties for dynamic values
- Minimize runtime style calculations

```css
/* Good: CSS class */
.tw-component-dynamic {
    background: var(--component-bg-color);
    color: var(--component-text-color);
}
```

```tsx
// Good: Set CSS variables
<div 
    className="tw-component-dynamic"
    style={{
        '--component-bg-color': backgroundColor,
        '--component-text-color': textColor,
    }}
/>
```

## Accessibility Patterns

### 1. Required Attributes

```tsx
// Always include accessibility attributes
<button
    aria-label={ariaLabel}
    aria-disabled={disabled}
    aria-busy={loading}
    disabled={disabled}
    role="button"  // If not using semantic HTML
>
```

### 2. Focus Management

```css
/* Consistent focus styles */
.tw-component-focus {
    @apply focus:tw-ring-2 focus:tw-ring-offset-2;
    @apply focus:tw-ring-primary focus:tw-ring-offset-background;
}
```

### 3. Screen Reader Support

```tsx
// Descriptive content for screen readers
{loading && <span className="tw-sr-only">Loading...</span>}

// Hide decorative elements
<div aria-hidden="true" role="presentation">
    {/* Decorative content */}
</div>
```

## Type Safety Patterns

### 1. Variant Types

```typescript
// Use string literal unions for variants
export type ComponentVariant = "primary" | "secondary" | "danger";

// Use Record types for mappings
const VARIANT_STYLES: Record<ComponentVariant, string> = {
    primary: "...",
    secondary: "...",
    danger: "...",
};
```

### 2. Props Interface

```typescript
// Extend HTML attributes for proper typing
export interface ComponentProps extends ComponentHTMLAttributes<HTMLElement> {
    variant?: ComponentVariant;
    size?: ComponentSize;
    children?: ReactNode;
}
```

### 3. Generic Components

```typescript
// For flexible components
interface GenericComponentProps<T = HTMLElement> extends ComponentHTMLAttributes<T> {
    as?: keyof JSX.IntrinsicElements;
    variant?: ComponentVariant;
}

export const GenericComponent = <T = HTMLElement>({
    as: Component = "div",
    ...props
}: GenericComponentProps<T>) => {
    return <Component {...props} />;
};
```

## Testing Patterns

### 1. Story Structure

```tsx
// ComponentName.stories.tsx
export const AllVariants: Story = {
    render: () => (
        <div className="tw-grid tw-grid-cols-3 tw-gap-4">
            {variants.map(variant => (
                <Component key={variant} variant={variant}>
                    {variant}
                </Component>
            ))}
        </div>
    ),
};

export const Interactive: Story = {
    render: () => {
        const [state, setState] = useState("default");
        return (
            <Component 
                onClick={() => setState("clicked")}
                variant={state as ComponentVariant}
            >
                {state}
            </Component>
        );
    },
};
```

### 2. Test Cases

```typescript
// Component.test.tsx
describe("Component", () => {
    it("renders with default props", () => {
        render(<Component>Test</Component>);
        expect(screen.getByRole("button")).toBeInTheDocument();
    });

    it("applies variant classes correctly", () => {
        render(<Component variant="primary">Test</Component>);
        expect(screen.getByRole("button")).toHaveClass("tw-component-primary");
    });

    it("handles accessibility attributes", () => {
        render(<Component disabled>Test</Component>);
        expect(screen.getByRole("button")).toHaveAttribute("aria-disabled", "true");
    });
});
```

## Migration Strategy

### 1. From MUI to Tailwind

1. **Identify component patterns** in existing MUI components
2. **Map styling properties** to Tailwind equivalents
3. **Preserve functionality** while updating styling approach
4. **Maintain API compatibility** where possible
5. **Update gradually** with feature parity

### 2. Gradual Adoption

```tsx
// Transition period: Allow both styling systems
const Component = ({ useTailwind = false, ...props }) => {
    if (useTailwind) {
        return <TailwindComponent {...props} />;
    }
    return <MUIComponent {...props} />;
};
```

## Best Practices Summary

### ✅ Do

- Use consistent naming conventions
- Extract styles to separate utility files
- Memoize expensive calculations
- Include comprehensive accessibility support
- Document component APIs with JSDoc
- Create reusable factory patterns
- Use TypeScript for type safety
- Follow the established file structure

### ❌ Don't

- Mix inline styles with Tailwind classes unnecessarily
- Create overly complex variant systems
- Forget accessibility attributes
- Skip performance optimizations
- Ignore TypeScript warnings
- Create inconsistent naming patterns
- Duplicate styling logic across components

## Example Implementation

See `Button.tsx` and `buttonStyles.ts` for a complete implementation following these patterns.

---

This pattern guide ensures consistency across all Tailwind components and provides a foundation for maintainable, performant, and accessible UI components.