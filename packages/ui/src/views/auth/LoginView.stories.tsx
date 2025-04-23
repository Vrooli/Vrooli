import { LoginView } from "./LoginView.js";

export default {
    title: "Views/Auth/LoginView",
    component: LoginView,
};

export function Default() {
    return (
        <LoginView display="Page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default login view.",
        },
    },
};
