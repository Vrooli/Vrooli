import { PageContainer } from "components/Page/Page.js";
import { SettingsFocusModesView } from "./SettingsFocusModesView.js";

export default {
    title: "Views/Settings/SettingsFocusModesView",
    component: SettingsFocusModesView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsFocusModesView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings focus modes view.",
        },
    },
};
