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
      // `.retired/` holds quarantined asset trees. Their imports are expected
      // to dangle — that is what retirement means — so running their tests
      // reports resolution failures for content nothing can reach.
      exclude: ["../library/.retired/**"],
    },
  };
});
