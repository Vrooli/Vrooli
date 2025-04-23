import { SignupView } from "./SignupView.js";

export default {
    title: "Views/Auth/SignupView",
    component: SignupView,
};

export function Default() {
    return (
        <SignupView display="Page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default signup view.",
        },
    },
};
