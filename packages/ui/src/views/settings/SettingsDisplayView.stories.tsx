import { PageContainer } from "components/Page/Page.js";
import { SettingsDisplayView } from "./SettingsDisplayView.js";

export default {
    title: "Views/Settings/SettingsDisplayView",
    component: SettingsDisplayView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsDisplayView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings display view.",
        },
    },
};
