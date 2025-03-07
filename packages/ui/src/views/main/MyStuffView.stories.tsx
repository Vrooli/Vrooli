import { MyStuffView } from "./MyStuffView.js";

export default {
    title: "Views/Main/MyStuffView",
    component: MyStuffView,
};

export function Default() {
    return (
        <MyStuffView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default my stuff view.",
        },
    },
};
