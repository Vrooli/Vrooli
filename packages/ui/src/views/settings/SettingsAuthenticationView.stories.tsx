import { PageContainer } from "components/Page/Page.js";
import { SettingsAuthenticationView } from "./SettingsAuthenticationView.js";

export default {
    title: "Views/Settings/SettingsAuthenticationView",
    component: SettingsAuthenticationView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsAuthenticationView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings authentication view.",
        },
    },
};
