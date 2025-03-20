import { useEffect } from "react";
import { COMMAND_PALETTE_ID, PubSub } from "../../../utils/pubsub.js";
import { CommandPalette } from "./CommandPalette.js";

export default {
    title: "Components/Navigation/CommandPalette",
    component: CommandPalette,
};

export function Default() {
    useEffect(() => {
        PubSub.get().publish("menu", { id: COMMAND_PALETTE_ID, isOpen: true });
    }, []);
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default find object dialog.",
        },
    },
};
