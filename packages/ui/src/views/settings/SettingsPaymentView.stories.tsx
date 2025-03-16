import { PageContainer } from "components/Page/Page.js";
import { SettingsPaymentView } from "./SettingsPaymentView.js";

export default {
    title: "Views/Settings/SettingsPaymentView",
    component: SettingsPaymentView,
};

export function Default() {
    return (
        <PageContainer>
            <SettingsPaymentView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings payment view.",
        },
    },
};
