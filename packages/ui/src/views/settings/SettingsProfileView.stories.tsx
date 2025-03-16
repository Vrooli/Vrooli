import { PageContainer } from "components/Page/Page.js";
import { SettingsProfileView } from "./SettingsProfileView.js";

export default {
    title: "Views/Settings/SettingsProfileView",
    component: SettingsProfileView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsProfileView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings profile view.",
        },
    },
};
