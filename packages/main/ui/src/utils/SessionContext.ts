import { Session } from "@local/shared";
import { createContext } from "react";

export const SessionContext = createContext<Session | undefined>(undefined);
