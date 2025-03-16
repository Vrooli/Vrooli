import { InboxView } from "./InboxView.js";

export default {
    title: "Views/Main/InboxView",
    component: InboxView,
};

export function Default() {
    return (
        <InboxView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default inbox view.",
        },
    },
};
