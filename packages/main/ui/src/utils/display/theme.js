import { createTheme } from "@mui/material";
const commonTheme = createTheme({
    components: {
        MuiButton: {
            defaultProps: {
                variant: "contained",
                color: "secondary",
            },
        },
        MuiTextField: {
            defaultProps: {
                variant: "outlined",
            },
        },
    },
});
const lightTheme = createTheme({
    ...commonTheme,
    palette: {
        mode: "light",
        primary: {
            light: "#4372a3",
            main: "#344eb5",
            dark: "#072c6a",
        },
        secondary: {
            light: "#4ae59d",
            main: "#16a361",
            dark: "#009b53",
        },
        background: {
            default: "#e9ebf1",
            paper: "#ebeff5",
            textPrimary: "#000000",
            textSecondary: "#6f6f6f",
        },
    },
});
const darkTheme = createTheme({
    ...commonTheme,
    palette: {
        mode: "dark",
        primary: {
            light: "#5f6a89",
            main: "#515774",
            dark: "#242930",
        },
        secondary: {
            light: "#5b99da",
            main: "#4372a3",
            dark: "#344eb5",
        },
        background: {
            default: "#181818",
            paper: "#2e2e2e",
            textPrimary: "#ffffff",
            textSecondary: "#c3c3c3",
        },
    },
});
export const themes = {
    "light": lightTheme,
    "dark": darkTheme,
};
//# sourceMappingURL=theme.js.map