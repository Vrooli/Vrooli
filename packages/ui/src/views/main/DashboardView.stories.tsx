import { LINKS } from "@vrooli/shared";
import { DashboardView } from "./DashboardView.js";

export default {
    title: "Views/Main/DashboardView",
    component: DashboardView,
};

export const Default = () => <DashboardView display="Page" />;
Default.parameters = {
    route: {
        path: LINKS.Home,
    },
};
