import type { StorybookConfig } from '@storybook/react-vite';

import { dirname, join } from "path";

/**
* This function is used to resolve the absolute path of a package.
* It is needed in projects that use Yarn PnP or are set up within a monorepo.
*/
function getAbsolutePath(value: string): any {
    return dirname(require.resolve(join(value, 'package.json')))
}
const config: StorybookConfig = {
    "stories": [
        "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"
    ],
    "addons": [
        {
            "name": getAbsolutePath('@storybook/addon-essentials'),
            "options": {
                "docs": false
            }
        },
        getAbsolutePath('@chromatic-com/storybook'),
        getAbsolutePath('@storybook/addon-interactions'),
        getAbsolutePath('@storybook/addon-a11y'),
    ],
    "framework": {
        "name": getAbsolutePath('@storybook/react-vite'),
        "options": {}
    },
    "core": {
        "disableTelemetry": true,
    },
    "typescript": {
        "configFile": "../tsconfig.stories.json"
    },
    viteFinal: async (config) => {
        // Mock Stripe to prevent network errors in Storybook
        config.resolve = config.resolve || {};
        config.resolve.alias = {
            ...config.resolve.alias,
            '@stripe/stripe-js': join(__dirname, './mocks/stripe.ts'),
            'socket.io-client': join(__dirname, './mocks/socket.ts'),
            // Use built shared package for Storybook to avoid JSON import issues
            '@vrooli/shared': join(__dirname, '../../shared/dist'),
        };
        return config;
    },
};
export default config;