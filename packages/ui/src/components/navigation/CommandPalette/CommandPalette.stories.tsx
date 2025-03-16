import { useEffect } from "react";
import { PubSub } from "../../../utils/pubsub.js";
import { CommandPalette } from "./CommandPalette.js";

export default {
    title: "Components/Navigation/CommandPalette",
    component: CommandPalette,
};

export function Default() {
    useEffect(() => {
        PubSub.get().publish("commandPalette");
    }, []);
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default find object dialog.",
        },
    },
};
