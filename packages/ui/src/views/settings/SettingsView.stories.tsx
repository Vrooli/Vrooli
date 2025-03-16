import { PageContainer } from "components/Page/Page.js";
import { SettingsView } from "./SettingsView.js";

export default {
    title: "Views/Settings/SettingsView",
    component: SettingsView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings view.",
        },
    },
};
