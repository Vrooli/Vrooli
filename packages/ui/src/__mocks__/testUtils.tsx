import { StyledEngineProvider, Theme, ThemeProvider, createTheme } from "@mui/material";
import { render as rtlRender } from "@testing-library/react";
import { SessionContext } from "../contexts/SessionContext";
import { ZIndexProvider } from "../contexts/ZIndexContext";
import { themes } from "../utils/display/theme";

const withFontSize = (theme: Theme, fontSize: number): Theme => createTheme({
    ...theme,
    typography: {
        fontSize,
    },
});

const withIsLeftHanded = (theme: Theme, isLeftHanded: boolean): Theme => createTheme({
    ...theme,
    isLeftHanded,
});

// Mock values or functions to simulate theme and session context
const defaultFontSize = 14;
const defaultIsLeftHanded = false;
const defaultTheme = "light"; // or 'dark', depending on your default

// Create a default theme object based on your customizations
const defaultCustomTheme = withIsLeftHanded(withFontSize(themes[defaultTheme], defaultFontSize), defaultIsLeftHanded);

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
                        <ZIndexProvider>
                            {children}
                        </ZIndexProvider>
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
