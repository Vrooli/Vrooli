import theme from "./tailwind.theme.json";
export default {
    content: ["./index.html", "./src/**/*.{ts,tsx}"],
    theme: {
        extend: theme
    },
    plugins: []
};
