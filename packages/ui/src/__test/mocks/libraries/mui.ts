// AI_CHECK: TYPE_SAFETY=fixed-mui-mock-any-types | LAST: 2025-06-28
import { vi } from "vitest";
import React from "react";

export const defaultPalette = {
  mode: "light",
  primary: {
    main: "#1976d2",
    light: "#42a5f5",
    dark: "#1565c0",
    contrastText: "#fff",
  },
  secondary: {
    main: "#dc004e",
    light: "#e33371",
    dark: "#9a0036",
    contrastText: "#fff",
  },
  error: {
    main: "#f44336",
    light: "#e57373",
    dark: "#d32f2f",
    contrastText: "#fff",
  },
  warning: {
    main: "#ff9800",
    light: "#ffb74d",
    dark: "#f57c00",
    contrastText: "rgba(0, 0, 0, 0.87)",
  },
  info: {
    main: "#2196f3",
    light: "#64b5f6",
    dark: "#1976d2",
    contrastText: "#fff",
  },
  success: {
    main: "#4caf50",
    light: "#81c784",
    dark: "#388e3c",
    contrastText: "rgba(0, 0, 0, 0.87)",
  },
  background: {
    default: "#fff",
    paper: "#fff",
    textPrimary: "rgba(0, 0, 0, 0.87)",
    textSecondary: "rgba(0, 0, 0, 0.6)",
  },
  text: {
    primary: "rgba(0, 0, 0, 0.87)",
    secondary: "rgba(0, 0, 0, 0.6)",
    disabled: "rgba(0, 0, 0, 0.38)",
  },
  action: {
    active: "rgba(0, 0, 0, 0.54)",
    hover: "rgba(0, 0, 0, 0.04)",
    selected: "rgba(0, 0, 0, 0.08)",
    disabled: "rgba(0, 0, 0, 0.26)",
    disabledBackground: "rgba(0, 0, 0, 0.12)",
  },
  divider: "rgba(0, 0, 0, 0.12)",
};

// Create a mock theme context that matches what components expect
const mockTheme = {
  palette: defaultPalette,
  spacing: (factor: number) => `${8 * factor}px`,
  shape: {
    borderRadius: 4,
  },
  transitions: {
    create: vi.fn((props: string | string[], options?: Record<string, unknown>) => "all 0.3s ease"),
  },
  breakpoints: {
    down: vi.fn((breakpoint: string) => `@media (max-width:${breakpoint})`),
    up: vi.fn((breakpoint: string) => `@media (min-width:${breakpoint})`),
    values: {
      xs: 0,
      sm: 600,
      md: 900,
      lg: 1200,
      xl: 1536,
    },
  },
  typography: {
    fontSize: 14,
    fontFamily: "\"Roboto\", \"Helvetica\", \"Arial\", sans-serif",
    fontWeightLight: 300,
    fontWeightRegular: 400,
    fontWeightMedium: 500,
    fontWeightBold: 700,
    caption: {
      fontSize: "0.75rem",
      fontFamily: "\"Roboto\", \"Helvetica\", \"Arial\", sans-serif",
      fontWeight: 400,
      lineHeight: 1.66,
      letterSpacing: "0.03333em",
    },
    body1: {
      fontSize: "1rem",
      fontFamily: "\"Roboto\", \"Helvetica\", \"Arial\", sans-serif",
      fontWeight: 400,
      lineHeight: 1.5,
      letterSpacing: "0.00938em",
    },
    subtitle1: {
      fontSize: "1rem",
      fontFamily: "\"Roboto\", \"Helvetica\", \"Arial\", sans-serif",
      fontWeight: 400,
      lineHeight: 1.75,
      letterSpacing: "0.00938em",
    },
  },
  zIndex: {
    modal: 1300,
    snackbar: 1400,
    tooltip: 1500,
  },
  mixins: {
    toolbar: {
      minHeight: 56,
    },
  },
  shadows: Array(25).fill("none"),
};

export const mockUseTheme = vi.fn(() => mockTheme);

// Mock styled components
const mockStyled = (Component: React.ComponentType<any>, options?: { shouldForwardProp?: (prop: string) => boolean }) => {
  return (stylesFn: (theme: typeof mockTheme) => Record<string, unknown>) => {
    // Return a component that forwards all props
    const StyledComponent = React.forwardRef((props: Record<string, unknown>, ref: React.Ref<unknown>) => {
      // Filter out any props that shouldForwardProp says not to forward
      let filteredProps = { ...props };
      if (options?.shouldForwardProp) {
        filteredProps = Object.keys(props).reduce((acc, key) => {
          if (options.shouldForwardProp(key)) {
            acc[key] = props[key];
          }
          return acc;
        }, {} as Record<string, unknown>);
      }
      return React.createElement(Component, { ...filteredProps, ref });
    });
    StyledComponent.displayName = `Styled(${Component.displayName || Component.name || "Component"})`;
    return StyledComponent;
  };
};

// Add options parameter support for styled
mockStyled.withConfig = () => mockStyled;

// Mock ThemeProvider that actually provides theme context
const MockThemeProvider = ({ children, theme = mockTheme }: { children: React.ReactNode; theme?: any }) => {
  // Create a context provider that makes the theme available to makeStyles
  return React.createElement(
    "div", 
    { 
      "data-testid": "mock-theme-provider",
      // Add theme to context by setting a data attribute that makeStyles can access
      "data-theme": JSON.stringify(theme),
    }, 
    children,
  );
};

// Mock makeStyles that returns a hook which returns empty classes
const mockMakeStyles = vi.fn((stylesFunction: any) => {
  return vi.fn(() => {
    // If stylesFunction is a function, call it with mock theme to generate class names
    if (typeof stylesFunction === "function") {
      const styles = stylesFunction(mockTheme);
      // Convert styles object to empty class names to prevent CSS-in-JS errors
      const classes: Record<string, string> = {};
      Object.keys(styles).forEach(key => {
        classes[key] = `mock-${key}`;
      });
      return classes;
    }
    return {};
  });
});

export const muiStylesMock = {
  useTheme: mockUseTheme,
  ThemeProvider: MockThemeProvider,
  createTheme: vi.fn((options: any) => ({ ...mockTheme, ...options })),
  makeStyles: mockMakeStyles,
  lighten: vi.fn((color: string, amount?: number) => {
    // Simple mock implementation - just return a slightly different color
    return color + "80"; // Add transparency
  }),
  darken: vi.fn((color: string, amount?: number) => {
    // Simple mock implementation
    return color + "CC"; // Add darker transparency
  }),
  alpha: vi.fn((color: string, value: number) => {
    // Simple mock implementation
    return color + Math.round(value * 255).toString(16).padStart(2, "0");
  }),
  styled: mockStyled,
};
