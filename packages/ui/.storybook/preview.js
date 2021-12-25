import React from 'react';
import { addDecorator, } from '@storybook/react';
import { themes } from 'utils';
import { Typography } from '@mui/material';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider } from '@mui/styles';
import { useDarkMode } from 'storybook-dark-mode';

const useStyles = (theme) => ({
    item: {
        backgroundColor: theme.palette.background.paper,
        color: theme.palette.text.primary,
        minWidth: '100vw',
        minHeight: '50vh',
        padding: '20px',
    },
    title: {
        textAlign: 'center',
    },
});

/**
 * Determines theme and layout of stories
 */
addDecorator((story) => {
    const theme = useDarkMode() ? themes.dark : themes.light;
    return (
        <ThemeProvider theme={theme}>
            <div style={useStyles(theme).item}>
                <Typography variant="h4" style={useStyles(theme).title}>{useDarkMode() ? 'Dark' : 'Light'} Theme</Typography>
                {story()}
            </div>
        </ThemeProvider>
    )
});

/**
 * Mocks react-router-dom
 */
addDecorator((story) => (
    <BrowserRouter>
        {story()}
    </BrowserRouter>
))

export const parameters = {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
        matchers: {
            color: /(background|color)$/i,
            date: /Date$/,
        },
    },
}