import { AboutView } from "./AboutView.js";

export default {
    title: "Views/AboutView",
    component: AboutView,
};

export function Default() {
    return (
        <AboutView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default about view.",
        },
    },
};
