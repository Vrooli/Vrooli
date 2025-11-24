# UX Mode

Focus on user experience improvements and accessibility. Make the application delightful to use, accessible to all users, and visually polished.

## Key Areas
- Accessibility: ARIA labels, keyboard navigation, screen reader support
- Responsive design: Test and optimize for all breakpoints
- User flows: Ensure smooth, intuitive paths for all operational targets
- Visual polish: Consistent spacing, typography, color usage
- Loading states: All async operations show appropriate feedback
- Error handling: Clear, helpful error messages with recovery paths
- Performance UX: Perceived performance, optimistic updates

## Guidelines
- Every interactive element must be keyboard accessible
- Use semantic HTML and proper ARIA attributes
- Test with screen readers if possible
- Ensure color contrast meets WCAG AA standards
- Add loading indicators for operations > 300ms
- Provide clear feedback for all user actions

## Success Criteria
- Accessibility score > 90%
- All operational targets have complete user flows
- Responsive across mobile, tablet, desktop
- Zero UX-breaking errors

## Recommended Tools
- browser-automation-studio (test user flows, validate accessibility)
- react-component-library (use consistent, accessible components)
- tidiness-manager (ensure code organization doesn't hinder UX work)
