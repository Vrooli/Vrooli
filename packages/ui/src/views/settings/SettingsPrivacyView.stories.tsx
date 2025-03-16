import { PageContainer } from "components/Page/Page.js";
import { SettingsPrivacyView } from "./SettingsPrivacyView.js";

export default {
    title: "Views/Settings/SettingsPrivacyView",
    component: SettingsPrivacyView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsPrivacyView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings privacy view.",
        },
    },
};
