import { useEffect } from "react";
import { ChatCrud } from "../views/objects/chat/ChatCrud.js";
import { type ViewDisplayType } from "../types.js";
import { PubSub } from "../utils/pubsub.js";
import { ELEMENT_IDS } from "../utils/consts.js";
import { SwarmDetailPanel } from "./swarm/SwarmDetailPanel.js";
import { useDrawerStore } from "../stores/drawerStore.js";

/**
 * Component that renders content for the right drawer.
 * Switches between ChatCrud and SwarmDetailPanel based on menu events.
 */
export function RightDrawerContent({ display }: { display: ViewDisplayType }) {
    const { rightDrawer, setRightDrawer, resetRightDrawer } = useDrawerStore();

    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("menu", (data) => {
            if (data.id === ELEMENT_IDS.RightDrawer && data.isOpen && data.data?.view) {
                setRightDrawer({
                    view: data.data.view,
                    chatId: data.data.chatId,
                    swarmConfig: data.data.swarmConfig,
                    swarmStatus: data.data.swarmStatus,
                    onApproveToolCall: data.data.onApproveToolCall,
                    onRejectToolCall: data.data.onRejectToolCall,
                    onStart: data.data.onStart,
                    onPause: data.data.onPause,
                    onResume: data.data.onResume,
                    onStop: data.data.onStop,
                });
            } else if (data.id === ELEMENT_IDS.RightDrawer && !data.isOpen) {
                // Reset to default view when closing
                resetRightDrawer();
            }
        });

        return () => unsubscribe();
    }, [setRightDrawer, resetRightDrawer]);

    switch (rightDrawer.view) {
        case "swarmDetail":
            return (
                <SwarmDetailPanel
                    swarmConfig={rightDrawer.swarmConfig || null}
                    swarmStatus={rightDrawer.swarmStatus}
                    chatId={rightDrawer.chatId}
                    onApproveToolCall={rightDrawer.onApproveToolCall}
                    onRejectToolCall={rightDrawer.onRejectToolCall}
                    onStart={rightDrawer.onStart}
                    onPause={rightDrawer.onPause}
                    onResume={rightDrawer.onResume}
                    onStop={rightDrawer.onStop}
                />
            );
        case "chat":
        default:
            return <ChatCrud display={display} isCreate={false} />;
    }
}
