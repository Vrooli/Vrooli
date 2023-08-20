import { createContext, useRef } from "react";

type ZIndexContextType = {
    getZIndex: () => number;
    releaseZIndex: () => void;
};

export const DEFAULT_Z_INDEX = 1500;

export const ZIndexContext = createContext<ZIndexContextType | undefined>(undefined);

export const ZIndexProvider = ({ children }) => {
    const stack = useRef<number[]>([]);

    const getZIndex = () => {
        const newZIndex = (stack.current.length > 0 ? stack.current[stack.current.length - 1] : DEFAULT_Z_INDEX) + 1;
        stack.current.push(newZIndex);
        console.log("getZIndex new list", stack.current);
        return newZIndex;
    };

    const releaseZIndex = () => {
        stack.current.pop();
        console.log("releaseZIndex new list", stack.current);
    };

    return (
        <ZIndexContext.Provider value={{ getZIndex, releaseZIndex }}>
            {children}
        </ZIndexContext.Provider>
    );
};
