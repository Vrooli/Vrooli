import { calculateOccurrences, FocusMode, Session } from "@local/shared";
import { getCurrentUser } from "utils/authentication/session";

/**
 * Finds all focus modes which are occuring right now
 * @param session Current session data
 * @returns Focus modes occuring right now
 */
export const currentFocusMode = async (session: Session | null | undefined): Promise<FocusMode[]> => {
    if (!session) return [];
    // Find current user
    const user = getCurrentUser(session);
    if (!user || !user.focusModes) return [];

    // Map each focus mode to a promise that resolves to either the focus mode or null
    const focusModePromises = user.focusModes.map(async (s) => {
        const now = new Date();
        // Find if the focus mode has an event right now
        const events = s.schedule ? await calculateOccurrences(s.schedule, now, new Date(now.getTime() + 1000)) : [];
        return events.length > 0 ? s : null;
    });

    // Resolve all promises
    const results = await Promise.all(focusModePromises);

    // Filter out the nulls (focus modes that are not occurring right now)
    return results.filter((s): s is FocusMode => s !== null);
};