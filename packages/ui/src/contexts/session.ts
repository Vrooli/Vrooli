import { type Session } from "@vrooli/shared";
import { createContext } from "react";

export const SessionContext = createContext<Session | undefined>(undefined);
