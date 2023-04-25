import { useEffect, useState } from "react";

/**
 * Lazy-loaded confetti
 */
export const AsyncConfetti = () => {
    const [Confetti, setConfetti] = useState<any>(null);

    useEffect(() => {
        const loadConfetti = async () => {
            const { default: loadedConfetti } = await import("react-confetti");
            setConfetti(() => loadedConfetti);
        };

        loadConfetti();
    }, []);

    if (!Confetti) {
        return null;
    }

    return (
        <Confetti
            initialVelocityY={-10}
            recycle={false}
            confettiSource={{
                x: 0,
                y: 40,
                w: window.innerWidth,
                h: 0,
            }}
        />
    );
};
