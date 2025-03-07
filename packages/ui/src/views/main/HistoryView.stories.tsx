import { HistoryView } from "./HistoryView.js";

export default {
    title: "Views/Main/HistoryView",
    component: HistoryView,
};

export function Default() {
    return (
        <HistoryView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default history view.",
        },
    },
};
