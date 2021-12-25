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

// // Add a global decorator that will render a dark background when the
// // "Color Scheme" knob is set to dark
// addDecorator((story) => {
//     // A knob for color scheme added to every story
//     const colorScheme = select('Color Scheme', ['light', 'dark'], 'light');

//     // Hook your theme provider with some knobs
//     return React.createElement(ThemeProvider, {
//         // A knob for theme added to every story
//         theme: select('Theme', Object.keys(themes), 'default'),
//         colorScheme,
//         children: [
//             React.createElement('style', {
//                 dangerouslySetInnerHTML: {
//                     __html: `html { ${colorScheme === 'dark' ? 'background-color: rgb(35,35,35);' : ''
//                         } }`
//                 }
//             }),
//             story()
//         ]
//     });
// })

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