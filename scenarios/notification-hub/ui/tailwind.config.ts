import { type Config } from "tailwindcss";
import fileViewerPreset from "@vrooli/file-viewer/tailwind-preset";
import theme from "./tailwind.theme.json";

/**
 * The theme colors are CSS variables holding hex values, which Tailwind cannot
 * split into channels, so `bg-app-primary/10` would silently generate no CSS at
 * all. Giving each color an opacity function makes every `/NN` modifier
 * (including `backdrop:` and `placeholder:` variants) resolve with `color-mix`.
 */
function withOpacity(value: string) {
  return ({ opacityValue }: { opacityValue?: string }) =>
    opacityValue === undefined ? value : `color-mix(in srgb, ${value} calc(${opacityValue} * 100%), transparent)`;
}

// Tailwind resolves color functions at build time; its published `Config` types only list strings.
const colors = Object.fromEntries(Object.entries(theme.colors).map(([name, value]) => [name, withOpacity(value)])) as unknown as typeof theme.colors;
const extend: typeof theme = { ...theme, colors };

export default {
  presets: [fileViewerPreset],
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    // BRIDGE (remove when the library emits no utility classes — see gate
    // catalog.utility-class). Some published RCL versions still ship Tailwind class strings; without
    // this glob their styling is purged from this app's bundle with no error.
    "./node_modules/@vrooli/react-component-library/dist/**/*.js",
    // The shared file viewer ships its Tailwind classes in dist.
    "./node_modules/@vrooli/file-viewer/dist/**/*.js"
  ],
  theme: {
    extend
  },
  plugins: []
} satisfies Config;
