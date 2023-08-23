import { createContext, useMemo, useRef } from "react";

type ZIndexContextType = {
    getZIndex: () => number;
    releaseZIndex: () => void;
};

export const DEFAULT_Z_INDEX = 1400;

export const ZIndexContext = createContext<ZIndexContextType | undefined>(undefined);

export const ZIndexProvider = ({ children }) => {
    const stack = useRef<number[]>([]);

    const getZIndex = () => {
        const newZIndex = (stack.current.length > 0 ? stack.current[stack.current.length - 1] : DEFAULT_Z_INDEX) + 5;
        stack.current.push(newZIndex);
        return newZIndex;
    };

    const releaseZIndex = () => {
        stack.current.pop();
    };

    const value = useMemo(() => ({ getZIndex, releaseZIndex }), []);

    return (
        <ZIndexContext.Provider value={value}>
            {children}
        </ZIndexContext.Provider>
    );
};
