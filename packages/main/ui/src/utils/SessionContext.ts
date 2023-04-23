import { Session } from "@local/consts";
import { createContext } from "react";

export const SessionContext = createContext<Session | undefined>(undefined);
