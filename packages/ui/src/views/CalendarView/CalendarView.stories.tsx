import { CalendarView } from "./CalendarView.js";

export default {
    title: "Views/CalendarView",
    component: CalendarView,
};

export function Default() {
    return (
        <CalendarView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default calendar view.",
        },
    },
};
