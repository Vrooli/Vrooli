import { createTheme, lighten } from "@mui/material";

// Define custom theme properties
declare module "@mui/material/styles/createPalette" {
    interface TypeBackground {
        textPrimary: string;
        textSecondary: string;
    }
}

// Define common theme options (button appearance, etc.)
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

const lightPalette = {
    mode: "light",
    primary: {
        light: "#4372a3",
        main: "#344eb5",
        dark: "#072c6a",
        contrastText: "#ffffff",
    },
    secondary: {
        light: "#4ae59d", //'#96d175'
        main: "#16a361", //'#42bd3a'
        dark: "#009b53", //'#367032'
        contrastText: "#ffffff",
    },
    background: {
        default: "#e9ebf1",
        paper: "#ebeff5",
        textPrimary: "#000000",
        textSecondary: "#6f6f6f",
    },
} as const;
const lightTheme = createTheme({
    ...commonTheme,
    palette: lightPalette,
    components: {
        // Override the default MuiTextField background color
        MuiTextField: {
            styleOverrides: {
                root: {
                    "& .MuiOutlinedInput-root": {
                        backgroundColor: lightPalette.background.paper,
                        color: lightPalette.background.textPrimary,
                    },
                },
            },
        },
        MuiButton: {
            variants: [
                {
                    props: { variant: "text" },
                    style: {
                        color: lightPalette.secondary.main,
                    },
                },
                {
                    props: { variant: "outlined" },
                    style: {
                        color: lightPalette.secondary.main,
                        borderColor: lightPalette.secondary.main,
                    },
                },
                {
                    props: { variant: "contained" },
                    style: {
                        backgroundColor: lightPalette.secondary.main,
                        color: lightPalette.secondary.contrastText,
                        "&:hover": {
                            backgroundColor: lighten(lightPalette.secondary.main, 0.1),
                        },
                    },
                },
            ],
        },
    },
});

// Dark theme
const darkPalette = {
    mode: "dark",
    primary: {
        light: "#5f6a89",
        main: "#515774",
        dark: "#242930",
        contrastText: "#ffffff",
    },
    secondary: {
        light: "#5b99da",
        main: "#4372a3",
        dark: "#344eb5",
        contrastText: "#ffffff",
    },
    background: {
        default: "#000000",
        paper: "#2e2e2e",
        textPrimary: "#ffffff",
        textSecondary: "#c3c3c3",
    },
} as const;
const darkTheme = createTheme({
    ...commonTheme,
    palette: darkPalette,
    components: {
        // Override the default MuiTextField background color
        MuiTextField: {
            styleOverrides: {
                root: {
                    "& .MuiOutlinedInput-root": {
                        backgroundColor: darkPalette.background.paper,
                        color: darkPalette.background.textPrimary,
                    },
                },
            },
        },
        MuiButton: {
            variants: [
                {
                    props: { variant: "text" },
                    style: {
                        color: darkPalette.secondary.main,
                    },
                },
                {
                    props: { variant: "outlined" },
                    style: {
                        color: darkPalette.secondary.main,
                        border: `1px solid ${darkPalette.secondary.main}`,
                    },
                },
                {
                    props: { variant: "contained" },
                    style: {
                        backgroundColor: darkPalette.secondary.main,
                        color: darkPalette.secondary.contrastText,
                        "&:hover": {
                            backgroundColor: lighten(darkPalette.secondary.main, 0.1),
                        },
                    },
                },
            ],
        },
    },
});

export const themes = {
    "light": lightTheme,
    "dark": darkTheme,
};
