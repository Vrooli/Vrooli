import { createTheme } from '@material-ui/core/styles';

const commonTheme = createTheme({
    components: {
        // Style sheet name ⚛️
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

const lightTheme = createTheme({
    ...commonTheme,
    palette: {
        mode: 'light',
        primary: {
            light: '#4446a2',
            main: '#344eb5',
            dark: '#002784',
        },
        secondary: {
            light: '#c8ffa5',
            main: '#96d175',
            dark: '#66a047',
        },
        background: {
            default: '#e9ebf1',
            paper: '#ffffff',
            textPrimary: '#000000',
            textSecondary: '#6f6f6f',
        },
    }
})

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