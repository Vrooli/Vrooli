import { uuid } from "@local/shared";
import { useEffect, useRef, useState } from "react";
import { useReward } from "react-rewards";
import { CelebrationType, PubSub } from "utils/pubsub";

const DEFAULT_CELEBRATION_TYPE: CelebrationType = "confetti";
const DEFAULT_EMOJI = "🎉";

export const Celebration = () => {
    const [state, setState] = useState<{
        celebrationType: CelebrationType;
        emojis: string[];
        id: string; // Used to prevent duplicate triggers
        isActive: boolean;
        positionStyle: React.CSSProperties;
    }>({
        celebrationType: DEFAULT_CELEBRATION_TYPE,
        emojis: [DEFAULT_EMOJI],
        id: uuid(),
        isActive: false,
        positionStyle: {},
    });
    const { reward } = useReward("rewardId", state.celebrationType, state.celebrationType === "emoji" ? { emoji: state.emojis } : undefined);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const lastIdRef = useRef<string | null>(null);

    useEffect(() => {
        const celebrationSub = PubSub.get().subscribe("celebration", (data) => {
            console.log("in celebration sub", data);
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }

            // Update the celebration type
            let celebrationType: CelebrationType = DEFAULT_CELEBRATION_TYPE;
            if (data?.celebrationType) celebrationType = data.celebrationType;
            else if (data?.emojis) celebrationType = "emoji";

            // Update the emoji
            let emojis = [DEFAULT_EMOJI];
            if (data?.emojis) emojis = data.emojis;

            // Position the span around the target element if targetId is provided
            let positionStyle: React.CSSProperties = {};
            if (data?.targetId) {
                const targetElement = document.getElementById(data.targetId);
                if (targetElement) {
                    const rect = targetElement.getBoundingClientRect();
                    positionStyle = {
                        position: "absolute",
                        top: rect.top + window.scrollY + rect.height / 2,
                        left: rect.left + window.scrollX + rect.width / 2,
                        transform: "translate(-50%, -50%)",
                        zIndex: 100000,
                    };
                }
            } else {
                // Position the span at the bottom center of the screen
                positionStyle = {
                    position: "fixed",
                    left: "50%",
                    bottom: "0px",
                    transform: "translateX(-50%)",
                    zIndex: 100000,
                };
            }

            setState({
                celebrationType,
                emojis,
                id: uuid(),
                isActive: true,
                positionStyle,
            });

            const duration = data?.duration || 5000;
            timeoutRef.current = setTimeout(() => {
                setState(s => ({ ...s, isActive: false }));
            }, duration);
        });

        return () => {
            console.log("cleaning up celebration");
            PubSub.get().unsubscribe(celebrationSub);
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    useEffect(() => {
        if (state.isActive && state.id !== lastIdRef.current) {
            lastIdRef.current = state.id;
            reward();
        }
    }, [reward, state]);

    return (
        <span id="rewardId" style={state.positionStyle} />
    );
};
