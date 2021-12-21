import React from 'react';

// Create decorator to wrap stories with MUI theme
import { addDecorator } from '@storybook/react';
import { ThemeProvider } from '@material-ui/core/styles';
import { themes } from 'utils';
import { Grid } from '@material-ui/core';
addDecorator((story) => (
    <div>
        <Grid>
            <Grid item xs={12}>
                Light theme: <ThemeProvider theme={themes.light}>{story()}</ThemeProvider>
            </Grid>
            <Grid item xs={12}>
                Dark theme: <ThemeProvider theme={themes.dark}>{story()}</ThemeProvider>
            </Grid>
        </Grid>
    </div>
));

export const parameters = {
  actions: { argTypesRegex: "^on[A-Z].*" },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
}