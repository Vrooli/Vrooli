import { PageContainer } from "components/Page/Page.js";
import { SettingsDataView } from "./SettingsDataView.js";

export default {
    title: "Views/Settings/SettingsDataView",
    component: SettingsDataView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsDataView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings data view.",
        },
    },
};
