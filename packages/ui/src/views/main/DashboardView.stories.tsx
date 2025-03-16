import { DashboardView } from "./DashboardView.js";

export default {
    title: "Views/Main/DashboardView",
    component: DashboardView,
};

export function Default() {
    return (
        <DashboardView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default dashboard view.",
        },
    },
};
