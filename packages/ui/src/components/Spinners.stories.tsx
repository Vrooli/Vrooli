import { useEffect } from "react";
import { ELEMENT_IDS } from "../utils/consts.js";
import { PubSub } from "../utils/pubsub.js";
import { FullPageSpinner } from "./Spinners.js";

const OPEN_DELAY_MS = 1000;

export default {
    title: "Components/Spinners",
    component: FullPageSpinner,
};

export function Loading() {
    useEffect(function openSpinnerEffect() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.FullPageSpinner, isOpen: true });
        }, OPEN_DELAY_MS);
    }, []);

    return (
        <FullPageSpinner />
    );
}
Loading.parameters = {
    docs: {
        description: {
            story: "Displays the active spinner.",
        },
    },
};

export function NotLoading() {
    useEffect(function closeSpinnerEffect() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.FullPageSpinner, isOpen: false });
        }, OPEN_DELAY_MS);
    }, []);

    return (
        <FullPageSpinner />
    );
}
NotLoading.parameters = {
    docs: {
        description: {
            story: "Displays the inactive spinner (shouldn't be visible).",
        },
    },
};
