import { ResetPasswordView } from "./ResetPasswordView.js";

export default {
    title: "Views/Auth/ResetPasswordView",
    component: ResetPasswordView,
};

export function Default() {
    return (
        <ResetPasswordView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default reset password view.",
        },
    },
};
