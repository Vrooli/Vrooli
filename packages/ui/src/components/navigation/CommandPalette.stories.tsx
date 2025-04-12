import { useEffect } from "react";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { CommandPalette } from "./CommandPalette.js";

export default {
    title: "Components/Navigation/CommandPalette",
    component: CommandPalette,
};

export function Default() {
    useEffect(() => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.CommandPalette, isOpen: true });
    }, []);
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default find object dialog.",
        },
    },
};
