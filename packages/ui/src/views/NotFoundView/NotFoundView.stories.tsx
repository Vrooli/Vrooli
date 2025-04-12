import { NotFoundView } from "./NotFoundView.js";

export default {
    title: "Views/NotFoundView",
    component: NotFoundView,
};

export function Default() {
    return (
        <NotFoundView />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default not found view.",
        },
    },
};
