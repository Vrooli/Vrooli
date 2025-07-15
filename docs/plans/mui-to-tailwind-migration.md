# MUI to Tailwind Migration Plan

## Overview
This document outlines the migration plan for replacing MUI components with custom Tailwind CSS components in the UI package. The migration will reduce bundle size, improve performance, and provide better design system consistency.

## Current State Analysis
- **Total MUI imports:** ~1,600+ across 960+ TypeScript/JSX files
- **Components already migrated:** 14 core components (Box, Typography, Button, Stack, etc.)
- **Remaining components:** 30+ components still using MUI

## Migration Strategy

### Phase 1: High Priority Components (10+ uses each)
**Goal:** Replace the most frequently used components to maximize impact

1. **TextField** (59 uses) 
   - Critical form input component
   - Note: TextInput component exists but needs investigation for compatibility
   - Priority: HIGHEST

2. **List/ListItem/ListItemText/ListItemIcon** (83 uses total)
   - Essential for navigation, menus, and data display
   - Should be built as a cohesive system
   - Priority: HIGHEST

3. **FormControl** (30 uses) + **FormLabel** (40 uses) + **FormHelperText** (7 uses)
   - Form field wrapper system
   - Should be designed together as unified form components
   - Priority: HIGH

4. **Chip** (19 uses)
   - Tag/chip component for labels and categories
   - Common UI pattern throughout app
   - Priority: HIGH

5. **InputAdornment** (17 uses)
   - Input icons and decorators (search icons, clear buttons, etc.)
   - Often used with TextField
   - Priority: HIGH

6. **Paper** (14 uses)
   - Elevated surface container
   - Simple styling component, quick win
   - Priority: HIGH

7. **Popover** (12 uses)
   - Floating positioned content
   - Complex positioning logic required
   - Priority: MEDIUM-HIGH

8. **LinearProgress** (11 uses)
   - Horizontal progress bars
   - Common loading indicator
   - Priority: MEDIUM-HIGH

### Phase 2: Medium Priority Components (5-10 uses each)
**Goal:** Address commonly used components for better coverage

9. **Menu/MenuItem** (24 uses total)
   - Dropdown menus and context menus
   - Complex interaction patterns

10. **Card/CardContent** (10 uses total)
    - Card layout components
    - Common content container pattern

11. **Avatar/AvatarGroup** (9 uses total)
    - User profile images and groups
    - Image handling and fallbacks needed

12. **Link** (8 uses)
    - Styled navigation links
    - Router integration considerations

13. **CircularProgress** (7 uses)
    - Loading spinners
    - Animation requirements

14. **Collapse** (7 uses)
    - Expandable content sections
    - Animation and transition logic

15. **Container** (6 uses)
    - Layout container with responsive max-width
    - Responsive design patterns

16. **Tabs/Tab** (6 uses total)
    - Tab navigation component
    - State management and keyboard navigation

17. **Autocomplete** (5 uses)
    - Searchable dropdown input
    - Complex search and filtering logic

### Phase 3: Lower Priority Components (1-4 uses each)
**Goal:** Complete the migration for full MUI removal

18. **Select** (4 uses) - Basic dropdown select
19. **InputLabel** (4 uses) - Input field labels  
20. **Input/InputBase** (5 uses total) - Base input components
21. **Fade/Slide** (7 uses total) - Animation transitions
22. **Popper** (3 uses) - Low-level positioning utility
23. **Skeleton** (2 uses) - Loading placeholder skeletons
24. **SwipeableDrawer** (2 uses) - Mobile navigation drawer
25. **Alert/Snackbar** (2 uses total) - Notification components

### Phase 4: Styling System Migration
**Goal:** Remove MUI theming and styling dependencies

26. **useTheme** (172 uses) - Replace with custom theme context
27. **styled** (101 uses) - Replace with Tailwind utilities
28. **@mui/material/styles** - Remove all MUI theming utilities

## Implementation Guidelines

### Component Requirements
- **Accessibility:** All components must meet WCAG 2.1 AA standards
- **TypeScript:** Full TypeScript support with proper prop interfaces
- **Responsive:** Mobile-first responsive design
- **Testing:** Unit tests and Storybook stories required
- **Documentation:** Clear prop documentation and usage examples

### Design System Consistency
- Follow existing Tailwind component patterns in `/components/`
- Use consistent naming conventions
- Maintain design token consistency (colors, spacing, typography)
- Ensure proper theme integration

### Migration Process for Each Component
1. **Analyze current usage** - Find all instances of the MUI component
2. **Design API** - Define props interface matching or improving MUI API
3. **Implement component** - Build with Tailwind CSS classes
4. **Add tests** - Unit tests and accessibility tests
5. **Create stories** - Storybook documentation
6. **Migrate usage** - Replace MUI imports gradually
7. **Test thoroughly** - Ensure no regressions

### Testing Strategy
- Use existing testcontainers setup for integration tests
- Focus on behavior testing, not visual testing
- Test keyboard navigation and screen reader compatibility
- Performance testing for complex components (List, Autocomplete)

## Success Metrics
- **Bundle size reduction:** Target 30-40% reduction in MUI-related bundle size
- **Performance improvement:** Faster initial load and render times
- **Developer experience:** Consistent API and better TypeScript support
- **Design consistency:** More cohesive visual design system

## Risk Mitigation
- **Gradual migration:** Migrate components one at a time to avoid breaking changes
- **Backward compatibility:** Keep MUI components working during transition
- **Thorough testing:** Comprehensive test coverage before removing MUI dependencies
- **Feature parity:** Ensure all existing functionality is preserved

## Timeline Estimate
- **Phase 1:** 3-4 weeks (high-impact components)
- **Phase 2:** 2-3 weeks (medium-impact components)  
- **Phase 3:** 1-2 weeks (low-impact components)
- **Phase 4:** 1-2 weeks (styling system migration)
- **Total:** 7-11 weeks for complete migration

## Notes
- Some components like TextField may have existing partial implementations (TextInput)
- Form components should be designed as a cohesive system
- Complex components like Autocomplete and Popover will require the most development time
- Consider using headless UI libraries (Radix, Headless UI) for complex interaction patterns

## Next Steps
1. Start with TextField migration (highest impact)
2. Investigate existing TextInput component compatibility
3. Design form component system architecture
4. Create implementation plan for List components
5. Set up component testing and documentation workflow