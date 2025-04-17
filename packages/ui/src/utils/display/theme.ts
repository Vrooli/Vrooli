import { createTheme, lighten } from "@mui/material";

// Define custom theme properties
declare module "@mui/material/styles/createPalette" {
    interface TypeBackground {
        textPrimary: string;
        textSecondary: string;
    }
}

export const BREAKPOINTS = {
    xs: 0,
    sm: 600,
    md: 900,
    lg: 1200,
    xl: 1536,
} as const;

export const drawerTransitionDuration = { enter: 300, exit: 300 };

// Define common theme options (button appearance, etc.)
const commonTheme = createTheme({
    breakpoints: {
        values: BREAKPOINTS,
    },
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
        MuiDrawer: {
            defaultProps: {
                transitionDuration: drawerTransitionDuration,
            },
        },
        MuiSwipeableDrawer: {
            defaultProps: {
                transitionDuration: drawerTransitionDuration,
            },
        },
    },
});

export const lightPalette = {
    mode: "light",
    primary: {
        light: "#4372a3",
        main: "#264983",
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
        default: "#c2cadd", // "#e9ebf1",
        paper: "#ebeff5",
        textPrimary: "#000000",
        textSecondary: "#6f6f6f",
    },
    divider: "rgba(0, 0, 0, 0.23)",
} as const;
const lightTheme = createTheme({
    ...commonTheme,
    palette: lightPalette,
    components: {
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
                            // eslint-disable-next-line no-magic-numbers
                            backgroundColor: lighten(lightPalette.secondary.main, 0.1),
                        },
                    },
                },
            ],
        },
        MuiIconButton: {
            defaultProps: {
                disableRipple: true, // GlobalStyles overrides highlighting behavior
            },
        },
        MuiTextField: {
            styleOverrides: {
                root: ({ theme }) => ({
                    '& .MuiOutlinedInput-root': {
                        color: lightPalette.background.textPrimary,
                        backgroundColor: theme.palette.background.paper,
                        borderRadius: theme.spacing(3),
                        '& fieldset': {
                            border: 'none',
                        },
                        '&:hover fieldset': {
                            border: 'none',
                        },
                        '&.Mui-focused fieldset': {
                            border: 'none',
                        },
                    },
                    '& label.Mui-focused': {
                        color: theme.palette.background.textSecondary,
                    },
                }),
            },
        },
    },
});

// Dark theme
export const darkPalette = {
    mode: "dark",
    primary: {
        light: "#5f6a89",
        main: "#2e3847",
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
    divider: "rgba(255, 255, 255, 0.23)",
} as const;
const darkTheme = createTheme({
    ...commonTheme,
    palette: darkPalette,
    components: {
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
                            // eslint-disable-next-line no-magic-numbers
                            backgroundColor: lighten(darkPalette.secondary.main, 0.1),
                        },
                    },
                },
            ],
        },
        MuiIconButton: {
            defaultProps: {
                disableRipple: true, // GlobalStyles overrides highlighting behavior
            },
        },
        MuiTextField: {
            styleOverrides: {
                root: ({ theme }) => ({
                    '& .MuiOutlinedInput-root': {
                        color: darkPalette.background.textPrimary,
                        backgroundColor: theme.palette.background.paper,
                        borderRadius: theme.spacing(3),
                        '& fieldset': {
                            border: 'none',
                        },
                        '&:hover fieldset': {
                            border: 'none',
                        },
                        '&.Mui-focused fieldset': {
                            border: 'none',
                        },
                    },
                    '& label.Mui-focused': {
                        color: theme.palette.background.textSecondary,
                    },
                }),
            },
        },
    },
});

export const themes = {
    "light": lightTheme,
    "dark": darkTheme,
};

export const DEFAULT_THEME = "dark";
