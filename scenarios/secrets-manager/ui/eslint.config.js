import tsPlugin from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

const tsRecommended = tsPlugin.configs.recommended;
const hooksRecommended = reactHooks.configs["recommended-latest"];
const refreshRecommended = reactRefresh.configs.vite;

export default [
  {
    files: ["**/*.{ts,tsx}", "*.{ts,tsx}"],
    ignores: ["dist"],
    languageOptions: {
      parser: tsParser,
      ecmaVersion: 2020,
      sourceType: "module",
      globals: {
        window: "readonly",
        document: "readonly",
        navigator: "readonly",
        console: "readonly"
      }
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh
    },
    rules: {
      ...(tsRecommended?.rules ?? {}),
      ...(hooksRecommended?.rules ?? {}),
      ...(refreshRecommended?.rules ?? {})
    }
  }
];
