import { PageContainer } from "components/Page/Page.js";
import { SettingsNotificationsView } from "./SettingsNotificationsView.js";

export default {
    title: "Views/Settings/SettingsNotificationsView",
    component: SettingsNotificationsView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsNotificationsView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings notifications view.",
        },
    },
};
