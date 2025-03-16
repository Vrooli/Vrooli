import { LandingView } from "./LandingView.js";

export default {
    title: "Views/Main/LandingView",
    component: LandingView,
};

export function Default() {
    return (
        <LandingView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default landing view.",
        },
    },
};
