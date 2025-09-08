# UI Visual Validation Protocol

## Critical Understanding

**Taking screenshots without READING them is useless.** The Read tool allows you to visually inspect images. You MUST use it to verify UI correctness.

## When UI Validation is Required

UI validation is MANDATORY for:
- Any scenario with a web interface
- Any scenario with user-facing components
- Before marking ANY UI-related PRD items complete
- After making changes that could affect layout
- When updating dependencies (CSS, JS libraries)

## Complete UI Validation Process

### 1. Setup Browserless
```bash
# Ensure browserless is running
vrooli resource browserless start
vrooli resource browserless health
```

### 2. Take Comprehensive Screenshots
```bash
# Cover all major views and states
resource-browserless screenshot http://localhost:[PORT]/ --output /tmp/ui-home.png
resource-browserless screenshot http://localhost:[PORT]/dashboard --output /tmp/ui-dashboard.png
resource-browserless screenshot http://localhost:[PORT]/settings --output /tmp/ui-settings.png
resource-browserless screenshot http://localhost:[PORT]/login --output /tmp/ui-login.png
resource-browserless screenshot http://localhost:[PORT]/error --output /tmp/ui-error.png

# Different viewport sizes (if responsive)
resource-browserless screenshot http://localhost:[PORT] --width 375 --height 667 --output /tmp/ui-mobile.png
resource-browserless screenshot http://localhost:[PORT] --width 768 --height 1024 --output /tmp/ui-tablet.png
resource-browserless screenshot http://localhost:[PORT] --width 1920 --height 1080 --output /tmp/ui-desktop.png
```

### 3. READ and Inspect Each Screenshot (CRITICAL!)
```bash
# You MUST use the Read tool to visually inspect EACH screenshot
Read /tmp/ui-home.png       # Look at the actual homepage
Read /tmp/ui-dashboard.png  # Check dashboard layout
Read /tmp/ui-settings.png   # Verify settings page
Read /tmp/ui-mobile.png     # Check mobile responsiveness
```

### 4. Visual Inspection Checklist

When reading screenshots, check for:

#### Layout Issues
- [ ] No overlapping elements
- [ ] Proper spacing and padding
- [ ] Aligned components
- [ ] Consistent margins
- [ ] No content cutoff

#### Visual Elements
- [ ] All images load properly
- [ ] Icons display correctly
- [ ] Fonts render properly
- [ ] Colors match design
- [ ] Buttons are visible

#### Content Display
- [ ] Text is readable (no truncation)
- [ ] Forms show all fields
- [ ] Tables display data correctly
- [ ] Lists are properly formatted
- [ ] Navigation menus work

#### Interactive States
- [ ] Hover states visible
- [ ] Active/selected states clear
- [ ] Disabled states grayed out
- [ ] Loading states display
- [ ] Error messages show

#### Responsive Design
- [ ] Mobile layout works
- [ ] Tablet layout works
- [ ] Desktop layout works
- [ ] No horizontal scroll
- [ ] Touch targets adequate size

### 5. Document Visual Findings
```markdown
## UI Visual Validation Results

### Screenshots Taken
- home.png: Homepage at 1920x1080
- dashboard.png: Dashboard view
- mobile.png: Mobile view at 375x667

### Visual Issues Found
1. **Navigation Menu**: Overlaps content on mobile
   - Screenshot: /tmp/ui-mobile.png
   - Impact: Blocks user interaction
   - Priority: HIGH

2. **Dashboard Cards**: Missing border on hover
   - Screenshot: /tmp/ui-dashboard.png
   - Impact: Reduced visual feedback
   - Priority: LOW

### Visual Confirmations
- ✅ Homepage layout correct
- ✅ Forms display all fields
- ✅ Colors match brand guidelines
- ✅ Images and icons load properly
```

## Before/After Comparison Protocol

For improvements, ALWAYS compare:

```bash
# BEFORE changes
resource-browserless screenshot http://localhost:[PORT] --output /tmp/before.png
Read /tmp/before.png  # Document current state

# Make your changes

# AFTER changes
resource-browserless screenshot http://localhost:[PORT] --output /tmp/after.png
Read /tmp/after.png  # Compare to before

# Document the comparison
## Visual Changes
### Improvements
- Fixed navigation overlap on mobile
- Added missing hover states
- Improved form field spacing

### Regressions (FIX IMMEDIATELY)
- Button colors changed unexpectedly
- Font size reduced unintentionally
```

## Common UI Issues to Watch For

1. **CSS Not Loading**: Page appears unstyled
2. **JavaScript Errors**: Interactive elements don't work
3. **Responsive Breaks**: Layout fails at certain sizes
4. **Asset 404s**: Images/fonts missing
5. **Z-index Issues**: Elements appear above/below incorrectly
6. **Animation Glitches**: Transitions broken
7. **Form Validation**: Error messages not showing
8. **Accessibility**: Contrast too low, text too small

## PRD Validation via Screenshots

NEVER mark these PRD items complete without screenshot verification:
- [ ] "Responsive design"
- [ ] "Beautiful UI"
- [ ] "Intuitive interface"
- [ ] "Mobile-friendly"
- [ ] "Accessible design"
- [ ] "Modern look and feel"
- [ ] "Consistent styling"
- [ ] "Professional appearance"

## Golden Rules

1. **No screenshot = No UI validation**
2. **No Read = No visual verification**
3. **UI broken = PRD items unchecked**
4. **Visual regression = STOP and fix**
5. **Document with screenshots**

## Quick UI Check Command
```bash
# One-liner for basic UI validation
resource-browserless screenshot http://localhost:[PORT] --output /tmp/quick-ui.png && Read /tmp/quick-ui.png
```

Remember: **Users see the UI, not your code.** A perfect backend with a broken UI is a failed product.