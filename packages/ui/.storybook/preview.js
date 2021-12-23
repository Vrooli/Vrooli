import React from 'react';
import { addDecorator, } from '@storybook/react';
import { themes } from 'utils';
import { Typography } from '@mui/material';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider } from '@mui/styles';

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
addDecorator((story) => (
    <div>
        <ThemeProvider theme={themes.light}>
            <div style={useStyles(themes.light).item}>
                <Typography variant="h4" style={useStyles(themes.light).title}>Light Theme</Typography>
                {story()}
            </div>
        </ThemeProvider>
        <ThemeProvider theme={themes.dark}>
            <div style={useStyles(themes.dark).item}>
                <Typography variant="h4" style={useStyles(themes.dark).title}>Dark Theme</Typography>
                {story()}
            </div>
        </ThemeProvider>
    </div>
));

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