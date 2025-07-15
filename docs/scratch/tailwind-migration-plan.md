# Gradual Migration from MUI to Tailwind CSS

## Phase 1: Setup and Configuration

### Install Tailwind CSS
```bash
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

### Configure Tailwind to work alongside MUI
In `tailwind.config.js`:

```javascript
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./packages/ui/src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      // Sync with your MUI theme colors
      colors: {
        primary: {
          50: '#e3f2fd',
          100: '#bbdefb', 
          // ... map your MUI theme colors
        },
        background: {
          default: 'var(--background-default)',
          paper: 'var(--background-paper)',
        },
        text: {
          primary: 'var(--text-primary)',
          secondary: 'var(--text-secondary)',
        }
      },
      fontSize: {
        // Dynamic font sizes that sync with your theme
        'dynamic-xs': ['var(--font-size-xs)', '1.25'],
        'dynamic-sm': ['var(--font-size-sm)', '1.375'],
        'dynamic-base': ['var(--font-size-base)', '1.5'],
        'dynamic-lg': ['var(--font-size-lg)', '1.75'],
        'dynamic-xl': ['var(--font-size-xl)', '1.75'],
      },
      spacing: {
        // Add spacing that matches MUI's spacing function
        '0.5': '0.125rem', // theme.spacing(0.5)
        '1': '0.25rem',    // theme.spacing(1)
        '2': '0.5rem',     // theme.spacing(2)
        // etc.
      }
    },
  },
  plugins: [],
  // Important: Add prefix to avoid conflicts during transition
  prefix: 'tw-',
  corePlugins: {
    // Disable Tailwind's CSS reset to avoid conflicts with MUI
    preflight: false,
  }
}
```

## Phase 2: Bridge Theme Systems

### Create CSS Custom Properties Bridge
Update your `App.tsx` to export theme values as CSS variables:

```typescript
export function useThemeCSSVariables(theme: Theme, fontSize: number) {
  useEffect(() => {
    const root = document.documentElement;
    
    // Colors
    root.style.setProperty('--background-default', theme.palette.background.default);
    root.style.setProperty('--background-paper', theme.palette.background.paper);
    root.style.setProperty('--text-primary', theme.palette.text.primary);
    root.style.setProperty('--text-secondary', theme.palette.text.secondary);
    root.style.setProperty('--primary-main', theme.palette.primary.main);
    root.style.setProperty('--primary-dark', theme.palette.primary.dark);
    root.style.setProperty('--primary-light', theme.palette.primary.light);
    
    // Dynamic font sizes based on MUI's typography
    const typography = theme.typography;
    root.style.setProperty('--font-size-xs', `${fontSize * 0.75}px`);
    root.style.setProperty('--font-size-sm', `${fontSize * 0.875}px`);
    root.style.setProperty('--font-size-base', `${fontSize}px`);
    root.style.setProperty('--font-size-lg', `${fontSize * 1.125}px`);
    root.style.setProperty('--font-size-xl', `${fontSize * 1.25}px`);
    
    // Spacing scale
    for (let i = 0; i <= 10; i++) {
      root.style.setProperty(`--spacing-${i}`, theme.spacing(i));
    }
  }, [theme, fontSize]);
}
```

### Create Theme-Aware Tailwind Classes
Create a utility to generate theme-aware Tailwind classes:

```typescript
// utils/tailwind-theme.ts
export function createTailwindThemeClasses(theme: Theme) {
  return {
    // Background colors
    bgDefault: 'tw-bg-background-default',
    bgPaper: 'tw-bg-background-paper',
    
    // Text colors  
    textPrimary: 'tw-text-text-primary',
    textSecondary: 'tw-text-text-secondary',
    
    // Dynamic font sizes
    textXs: 'tw-text-dynamic-xs',
    textSm: 'tw-text-dynamic-sm', 
    textBase: 'tw-text-dynamic-base',
    textLg: 'tw-text-dynamic-lg',
    textXl: 'tw-text-dynamic-xl',
    
    // Common patterns
    buttonPrimary: 'tw-bg-primary-main tw-text-white tw-px-4 tw-py-2 tw-rounded hover:tw-bg-primary-dark',
    cardContainer: 'tw-bg-background-paper tw-rounded-lg tw-shadow-md tw-p-4',
  };
}
```

## Phase 3: Migration Strategy

### Start with Simple Components
Begin migration with leaf components (no children components):

1. **Buttons** - Start with simple buttons
2. **Typography** - Text components  
3. **Icons** - Icon containers
4. **Simple containers** - Basic Box/div wrappers

### Create Hybrid Components
During transition, create components that can use both systems:

```typescript
interface HybridButtonProps {
  useTailwind?: boolean;
  children: React.ReactNode;
  onClick?: () => void;
  variant?: 'primary' | 'secondary';
}

export function HybridButton({ useTailwind = false, variant = 'primary', ...props }: HybridButtonProps) {
  if (useTailwind) {
    const { buttonPrimary } = createTailwindThemeClasses(useTheme());
    return (
      <button 
        className={variant === 'primary' ? buttonPrimary : 'tw-bg-gray-200 tw-text-gray-800 tw-px-4 tw-py-2 tw-rounded'}
        {...props}
      />
    );
  }
  
  return <Button variant="contained" color={variant} {...props} />;
}
```

### Component Migration Checklist
For each component migration:

- [ ] Replace MUI imports with Tailwind classes
- [ ] Ensure theme colors are respected via CSS variables
- [ ] Maintain responsive behavior
- [ ] Test in both light and dark modes
- [ ] Verify font size changes work correctly
- [ ] Update any custom styled components
- [ ] Add TypeScript types for new props

## Phase 4: Benefits During Transition

### Performance Improvements
- **Bundle size**: Tailwind classes are purged, only used styles included
- **Runtime performance**: No JavaScript theme calculations
- **Development speed**: No component re-renders when theme changes

### Maintenance Benefits  
- **Consistency**: Utility classes enforce design system
- **Debugging**: Easier to inspect styles in DevTools
- **Team onboarding**: Developers familiar with CSS can contribute

## Phase 5: Gradual Rollout Plan

### Week 1-2: Foundation
- [ ] Install and configure Tailwind
- [ ] Set up CSS variables bridge
- [ ] Create theme utility functions
- [ ] Migrate 2-3 simple components

### Week 3-4: Core Components
- [ ] Migrate button components
- [ ] Migrate typography components
- [ ] Migrate basic layout containers

### Week 5-8: Complex Components
- [ ] Migrate form components
- [ ] Migrate navigation components
- [ ] Migrate dialog/modal components

### Week 9-12: Final Migration
- [ ] Migrate remaining MUI components
- [ ] Remove MUI dependencies
- [ ] Enable Tailwind preflight
- [ ] Clean up CSS variables bridge
- [ ] Update documentation

## Best Practices

1. **Keep MUI theme as source of truth** during transition
2. **Test thoroughly** - both visual and functional testing
3. **Maintain accessibility** - ensure ARIA attributes are preserved
4. **Use component feature flags** to control rollout
5. **Document migration decisions** for team reference
6. **Consider creating a design system** with Tailwind utilities

## Potential Challenges

- **Bundle size temporarily increases** (both systems loaded)
- **Styling conflicts** between MUI and Tailwind
- **Team learning curve** for Tailwind syntax
- **Responsive breakpoint differences** between systems
- **Animation/transition differences**

## Success Metrics

- [ ] Bundle size reduction after complete migration
- [ ] Development velocity improvements
- [ ] Reduced runtime performance overhead
- [ ] Improved design consistency
- [ ] Easier maintenance and updates 