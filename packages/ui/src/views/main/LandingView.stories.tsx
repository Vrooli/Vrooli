import { loggedOutSession } from "../../__test/storybookConsts.js";
import { LandingView } from "./LandingView.js";

export default {
    title: "Views/Main/LandingView",
    component: LandingView,
};

// You can only ever visit the landing view if you are logged out
export function LoggedOut() {
    return (
        <LandingView display="page" />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};
