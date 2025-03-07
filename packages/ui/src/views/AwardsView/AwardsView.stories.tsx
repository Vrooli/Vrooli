import { AwardsView } from "./AwardsView.js";

export default {
    title: "Views/AwardsView",
    component: AwardsView,
};

export function Default() {
    return (
        <AwardsView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default awards view.",
        },
    },
};
