import { calculateOccurrences } from "@local/utils";
import { getCurrentUser } from "../authentication/session";
export const currentFocusMode = (session) => {
    if (!session)
        return [];
    const user = getCurrentUser(session);
    if (!user)
        return [];
    return user.focusModes?.filter((s) => {
        const now = new Date();
        const events = s.schedule ? calculateOccurrences(s.schedule, now, new Date(now.getTime() + 1000)) : [];
        return events.length > 0;
    }) ?? [];
};
//# sourceMappingURL=scheduleTools.js.map