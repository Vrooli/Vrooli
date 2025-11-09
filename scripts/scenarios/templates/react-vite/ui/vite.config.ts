import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd());

  const uiPort = process.env.UI_PORT;
  if (!uiPort) {
    throw new Error("UI_PORT environment variable is required. Run the scenario through the Vrooli lifecycle so it is provided automatically.");
  }

  const rawApiBase = env.VITE_API_BASE_URL || process.env.VITE_API_BASE_URL;
  if (!rawApiBase) {
    throw new Error(
      "VITE_API_BASE_URL must be defined (e.g., http://localhost:<API_PORT>/api/v1). The lifecycle should export it before running dev/build commands."
    );
  }

  return {
    plugins: [react()],
    base: './',
    server: {
      host: true,
      port: Number(uiPort)
    }
  };
});
