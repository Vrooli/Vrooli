import { jsx as _jsx } from "react/jsx-runtime";
import { useEffect, useState } from "react";
export const AsyncConfetti = () => {
    const [Confetti, setConfetti] = useState(null);
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
    return (_jsx(Confetti, { initialVelocityY: -10, recycle: false, confettiSource: {
            x: 0,
            y: 40,
            w: window.innerWidth,
            h: 0,
        } }));
};
//# sourceMappingURL=AsyncConfett.js.map