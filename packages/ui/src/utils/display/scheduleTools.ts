import { Session, UserSchedule } from "@shared/consts";
import { getCurrentUser } from "utils/authentication"

/**
 * Finds all schedules which are occuring right now
 * @param session Current session data
 * @returns Schedules occuring right now
 */
export const currentSchedules = (session: Session): UserSchedule[] => {
    // Find current user
    const user = getCurrentUser(session);
    if (!user) return [];
    // Find schedules occuring right now. More explicitly, schedules which:
    // - Event start is in the past, and event end is in the future, 
    // - OR recurring start is in the past, and recurring end is in the future
    return user.schedules?.filter((s) => {
        const now = new Date();
        return (
            (s.eventStart <= now && s.eventEnd >= now) ||
            (s.recurring && s.recurrStart <= now && s.recurrEnd >= now)
        );
    }) ?? [];
}