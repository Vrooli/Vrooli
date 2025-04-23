import { ForgotPasswordView } from "./ForgotPasswordView.js";

export default {
    title: "Views/Auth/ForgotPasswordView",
    component: ForgotPasswordView,
};

export function Default() {
    return (
        <ForgotPasswordView display="Page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default forgot password view.",
        },
    },
};
