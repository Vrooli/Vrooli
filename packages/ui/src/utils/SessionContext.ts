import { Session } from "@shared/consts";
import { createContext } from "react";

export const SessionContext = createContext<Session | undefined>(undefined);