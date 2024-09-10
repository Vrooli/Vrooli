import { DUMMY_ID, Session, VALYXA_ID, noop } from "@local/shared";
import { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieMatchingChat } from "utils/cookies";

export const SessionContext = createContext<Session | undefined>(undefined);

type ZIndexContextType = {
    getZIndex: () => number;
    releaseZIndex: () => unknown;
};

export const DEFAULT_Z_INDEX = 1000;
const Z_INDEX_INCREMENT = 5;

export const ZIndexContext = createContext<ZIndexContextType | undefined>(undefined);

export function ZIndexProvider({ children }) {
    const stack = useRef<number[]>([]);

    function getZIndex() {
        const newZIndex = (stack.current.length > 0 ? stack.current[stack.current.length - 1] : DEFAULT_Z_INDEX) + Z_INDEX_INCREMENT;
        stack.current.push(newZIndex);
        return newZIndex;
    }

    function releaseZIndex() {
        stack.current.pop();
    }

    const value = useMemo(() => ({ getZIndex, releaseZIndex }), []);

    return (
        <ZIndexContext.Provider value={value}>
            {children}
        </ZIndexContext.Provider>
    );
}

type ActiveChatContext = {
    chatId: string;
    setChatId: (chatId: string) => unknown;
}

export const ActiveChatContext = createContext<ActiveChatContext>({
    chatId: DUMMY_ID,
    setChatId: noop,
});

export function ActiveChatProvider({ children }) {
    const session = useContext(SessionContext);
    const { id: userId } = getCurrentUser(session);

    const [chatId, setChatId] = useState<string>(userId ? (getCookieMatchingChat([userId, VALYXA_ID]) ?? DUMMY_ID) : DUMMY_ID);

    useEffect(function findChatIdEffect() {
        if (chatId) return;
        if (!userId) return;
        const newChatId = getCookieMatchingChat([userId, VALYXA_ID]);
        if (newChatId) setChatId(newChatId);
    }, [chatId, userId]);

    const value = useMemo(() => ({ chatId, setChatId }), [chatId]);

    return (
        <ActiveChatContext.Provider value={value}>
            {children}
        </ActiveChatContext.Provider>
    );
}
