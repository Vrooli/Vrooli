import { createTheme } from '@material-ui/core/styles';

// Define custom theme properties
declare module '@material-ui/core/styles/createPalette' {
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
                variant: 'contained',
                color: 'secondary',
            },
        },
        MuiTextField: {
            defaultProps: {
                variant: 'outlined'
            },
        },
    },
});

// Light theme
const lightTheme = createTheme({
    ...commonTheme,
    palette: {
        mode: 'light',
        primary: {
            light: '#3275af',
            main: '#344eb5',
            dark: '#002784',
        },
        secondary: {
            light: '#96d175',
            main: '#42bd3a',
            dark: '#367032'
        },
        background: {
            default: '#e9ebf1',
            paper: '#ffffff',
            textPrimary: '#000000',
            textSecondary: '#6f6f6f',
        },
    }
})

// Dark theme
const darkTheme = createTheme({
    ...commonTheme,
    palette: {
        mode: 'dark',
        primary: {
            light: '#39676d',
            main: '#073c42',
            dark: '#00171b',
        },
        secondary: {
            light: '#b5ffec',
            main: '#83d1ba',
            dark: '#52a08a',
        },
        background: {
            default: '#000000',
            paper: '#212121',
            textPrimary: '#ffffff',
            textSecondary: '#c3c3c3',
        },
    }
})

export const themes = {
    'light': lightTheme,
    'dark': darkTheme
}