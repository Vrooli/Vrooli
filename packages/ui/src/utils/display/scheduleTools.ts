import { FocusMode, Session } from "@shared/consts";
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
        return (
            (s.eventStart <= now && s.eventEnd >= now) ||
            (s.recurring && s.recurrStart <= now && s.recurrEnd >= now)
        );
    }) ?? [];
}