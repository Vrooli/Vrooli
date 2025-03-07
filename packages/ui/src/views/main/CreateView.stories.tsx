import { PageContainer } from "components/Page/Page.js";
import { CreateView } from "./CreateView.js";

export default {
    title: "Views/Main/CreateView",
    component: CreateView,
};

export function Default() {
    return (
        <PageContainer>
            <CreateView display="page" />
        </PageContainer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default create view.",
        },
    },
};
