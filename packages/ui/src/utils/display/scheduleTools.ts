import { FocusMode, Session } from "@shared/consts";
import { calculateOccurrences } from "@shared/utils";
import { getCurrentUser } from "utils/authentication/session";

/**
 * Finds all focus modes which are occuring right now
 * @param session Current session data
 * @returns Focus modes occuring right now
 */
export const currentFocusMode = (session: Session | null | undefined): FocusMode[] => {
    if (!session) return [];
    // Find current user
    const user = getCurrentUser(session);
    if (!user) return [];
    // Find focus modes occuring right now
    return user.focusModes?.filter((s) => {
        const now = new Date();
        // Find if the focus mode has an event right now
        const events = s.schedule ? calculateOccurrences(s.schedule, now, new Date(now.getTime() + 1000)) : [];
        return events.length > 0;
    }) ?? [];
}