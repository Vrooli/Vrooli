import { createContext, useState } from "react";

type ZIndexContextType = {
    getZIndex: () => number;
    releaseZIndex: () => void;
};

export const ZIndexContext = createContext<ZIndexContextType | undefined>(undefined);

export const ZIndexProvider = ({ children }) => {
    const [stack, setStack] = useState<number[]>([]);

    const getZIndex = () => {
        const newZIndex = (stack.length > 0 ? stack[stack.length - 1] : 1000) + 1;
        setStack(prevStack => [...prevStack, newZIndex]);
        return newZIndex;
    };

    const releaseZIndex = () => {
        setStack(prevStack => prevStack.slice(0, prevStack.length - 1));
    };

    return (
        <ZIndexContext.Provider value={{ getZIndex, releaseZIndex }}>
            {children}
        </ZIndexContext.Provider>
    );
};
