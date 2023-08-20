import { createContext, useRef } from "react";

type ZIndexContextType = {
    getZIndex: () => number;
    releaseZIndex: () => void;
};

export const ZIndexContext = createContext<ZIndexContextType | undefined>(undefined);

export const ZIndexProvider = ({ children }) => {
    const stack = useRef<number[]>([]);

    const getZIndex = () => {
        const newZIndex = (stack.current.length > 0 ? stack.current[stack.current.length - 1] : 1000) + 1;
        stack.current.push(newZIndex);
        return newZIndex;
    };

    const releaseZIndex = () => {
        stack.current.pop();
    };

    return (
        <ZIndexContext.Provider value={{ getZIndex, releaseZIndex }}>
            {children}
        </ZIndexContext.Provider>
    );
};
