import { defineConfig, type UserConfig } from "vite";

import baseConfig from "./vite.config";

// Library contracts live one directory above the UI workspace. Keep their
// resolver, asset-stamp, and provider setup identical to the application
// tests, while giving the dedicated library command its own discovery scope.
export default defineConfig((env) => {
  const config = baseConfig(env) as UserConfig;
  return {
    ...config,
    test: {
      ...config.test,
      include: ["../library/**/*.{test,spec}.{ts,tsx}"],
    },
  };
});
