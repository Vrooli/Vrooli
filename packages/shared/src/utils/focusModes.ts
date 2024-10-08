import { ActiveFocusMode, FocusMode, FocusModeStopCondition, Schedule } from "../api/generated/graphqlTypes";
import { calculateOccurrences } from "./schedules";

/**
 * Finds which focus modes are active for a given time frame, based on the focus 
 * modes' schedules. This can be overridden by manually setting the focus mode
 * @param focusModes The user's focus modes.
 * @param startDate The start of the time frame.
 * @param endDate The end of the time frame.
 * @returns An array of focus modes that are active for the given time frame.
 */
export const getFocusModesFromOccurrences = async (
    focusModes: FocusMode[],
    startDate: Date,
    endDate: Date,
): Promise<FocusMode[]> => {
    // Map each focus mode to a promise that resolves to its occurrences
    const occurrencesPromises = focusModes.map(async (focusMode) => {
        return focusMode.schedule ? await calculateOccurrences(focusMode.schedule as Schedule, startDate, endDate) : [];
    });

    // Resolve all promises
    const occurrences = await Promise.all(occurrencesPromises);

    // Filter the focus modes that have occurrences in the time frame
    const activeFocusModes = focusModes.filter((_, index) => {
        const occurrence = occurrences[index];
        return occurrence && occurrence.length > 0;
    });
    return activeFocusModes;
};

/**
 * Finds the actual active focus mode
 * @param currentlyActive The focus mode that is currently active, which 
 * may be expired
 * @param focusModes The user's focus modes
 * @returns The focus mode that should be active
 */
export const getActiveFocusMode = async (
    currentlyActive: ActiveFocusMode | null | undefined,
    focusModes: FocusMode[],
): Promise<ActiveFocusMode | null> => {
    // If there is an active focus mode
    if (currentlyActive) {
        // If the focus mode must be manually switched, then return it
        if (currentlyActive.stopCondition === FocusModeStopCondition.Never) {
            return currentlyActive;
        }
        // If the focus mode is set to stop at a certain time, then check if that time has passed
        else if (currentlyActive.stopCondition === FocusModeStopCondition.AfterStopTime && currentlyActive.stopTime) {
            if (new Date(currentlyActive.stopTime) > new Date()) {
                return currentlyActive;
            }
        }
    }
    // Get the focus modes that are active according to their schedules
    const activeFocusModes = await getFocusModesFromOccurrences(focusModes, new Date(), new Date(new Date().getTime() + 1000 * 60)); // Add 1 minute buffer just in case
    // If there are no active focus modes and currently active is set to NextBegins, then return 
    // currently active
    if (activeFocusModes.length === 0 && currentlyActive && currentlyActive.stopCondition === FocusModeStopCondition.NextBegins) {
        return currentlyActive;
    }
    // If there is at least one active focus mode, then return the first one. 
    // Otherwise, return null
    if (activeFocusModes.length > 0) {
        return {
            __typename: "ActiveFocusMode",
            mode: activeFocusModes[0]!,
            stopCondition: FocusModeStopCondition.Automatic,
        };
    }
    // If there is at least one focus mode in the list, then return the first one.
    // Otherwise, return null
    return focusModes.length > 0 ? {
        __typename: "ActiveFocusMode",
        mode: focusModes[0]!,
        stopCondition: FocusModeStopCondition.Automatic,
    } : null;
};
