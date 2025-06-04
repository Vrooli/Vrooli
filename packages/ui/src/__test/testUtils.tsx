import { StyledEngineProvider, type Theme, ThemeProvider, createTheme } from "@mui/material";
import { render as rtlRender } from "@testing-library/react";
import { SessionContext } from "../contexts/session.js";
import { DEFAULT_THEME, themes } from "../utils/display/theme.js";

function withFontSize(theme: Theme, fontSize: number): Theme {
    return createTheme({
        ...theme,
        typography: {
            fontSize,
        },
    });
}

function withIsLeftHanded(theme: Theme, isLeftHanded: boolean): Theme {
    return createTheme({
        ...theme,
        isLeftHanded,
    });
}

// Mock values or functions to simulate theme and session context
const defaultFontSize = 14;
const defaultIsLeftHanded = false;

// Create a default theme object based on your customizations
const defaultCustomTheme = withIsLeftHanded(withFontSize(themes[DEFAULT_THEME], defaultFontSize), defaultIsLeftHanded);

function render(ui, {
    theme = defaultCustomTheme,
    session = undefined, // Default session value
    ...renderOptions
} = {}) {
    function Wrapper({ children }) {
        return (
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={theme}>
                    <SessionContext.Provider value={session}>
                        {children}
                    </SessionContext.Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );
    }

    return rtlRender(ui, { wrapper: Wrapper, ...renderOptions });
}

// Re-export everything
export * from "@testing-library/react";
// Override the render method
export { render };
