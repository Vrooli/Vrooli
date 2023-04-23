import { FocusModeStopCondition } from "@local/consts";
export const calculateOccurrences = (schedule, timeframeStart, timeframeEnd) => {
    const timeframeDuration = timeframeEnd.getTime() - timeframeStart.getTime();
    if (timeframeDuration > 1000 * 60 * 60 * 24 * 365) {
        console.error("calculateOccurrences time frame to large", { trace: "0432", timeframeStart, timeframeEnd });
        return [];
    }
    const occurrences = [];
    const startTime = new Date(schedule.startTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    const endTime = new Date(schedule.endTime.toLocaleString("en-US", { timeZone: schedule.timezone }));
    const duration = endTime.getTime() - startTime.getTime();
    for (const recurrence of schedule.recurrences) {
        let currentStartTime = startTime;
        const maxOccurrences = 5000;
        let occurrenceCount = 0;
        while (currentStartTime <= timeframeEnd && occurrences.length < maxOccurrences) {
            const currentEndTime = new Date(currentStartTime.getTime() + duration);
            const prevStartTime = currentStartTime;
            occurrenceCount++;
            let isException = false;
            for (const exception of schedule.exceptions) {
                if (Math.abs(exception.originalStartTime.getTime() - currentStartTime.getTime()) < 60000) {
                    isException = true;
                    if (exception.newStartTime || exception.newEndTime) {
                        occurrences.push({
                            start: exception.newStartTime ? new Date(exception.newStartTime.toLocaleString("en-US", { timeZone: schedule.timezone })) : currentStartTime,
                            end: exception.newEndTime ? new Date(exception.newEndTime.toLocaleString("en-US", { timeZone: schedule.timezone })) : currentEndTime,
                        });
                    }
                    break;
                }
            }
            if (!isException) {
                occurrences.push({
                    start: currentStartTime,
                    end: currentEndTime,
                });
            }
            switch (recurrence.recurrenceType) {
                case "Daily":
                    currentStartTime = new Date(currentStartTime.getTime() + recurrence.interval * 24 * 60 * 60 * 1000);
                    break;
                case "Weekly":
                    if (recurrence.dayOfWeek) {
                        const dayOfWeek = recurrence.dayOfWeek;
                        const currentDayOfWeek = currentStartTime.getDay();
                        const daysUntilNextOccurrence = (dayOfWeek - currentDayOfWeek + 7) % 7 || recurrence.interval * 7;
                        currentStartTime = new Date(currentStartTime.getTime() + daysUntilNextOccurrence * 24 * 60 * 60 * 1000);
                    }
                    else {
                        currentStartTime = new Date(currentStartTime.getTime() + recurrence.interval * 7 * 24 * 60 * 60 * 1000);
                    }
                    break;
                case "Monthly":
                    if (recurrence.dayOfMonth) {
                        const dayOfMonth = recurrence.dayOfMonth;
                        const currentMonth = currentStartTime.getMonth();
                        const nextOccurrence = new Date(currentStartTime);
                        nextOccurrence.setMonth(currentMonth + recurrence.interval);
                        nextOccurrence.setDate(dayOfMonth);
                        if (nextOccurrence <= currentStartTime) {
                            nextOccurrence.setMonth(currentMonth + 2 * recurrence.interval);
                        }
                        currentStartTime = nextOccurrence;
                    }
                    else {
                        const newMonth = currentStartTime.getMonth() + recurrence.interval;
                        currentStartTime = new Date(currentStartTime.setMonth(newMonth));
                    }
                    break;
                case "Yearly":
                    if (recurrence.month && recurrence.dayOfMonth) {
                        const nextOccurrence = new Date(currentStartTime);
                        nextOccurrence.setFullYear(currentStartTime.getFullYear() + recurrence.interval);
                        nextOccurrence.setMonth(recurrence.month);
                        nextOccurrence.setDate(recurrence.dayOfMonth);
                        if (nextOccurrence <= currentStartTime) {
                            nextOccurrence.setFullYear(currentStartTime.getFullYear() + 2 * recurrence.interval);
                        }
                        currentStartTime = nextOccurrence;
                    }
                    else {
                        const newYear = currentStartTime.getFullYear() + recurrence.interval;
                        currentStartTime = new Date(currentStartTime.setFullYear(newYear));
                    }
                    break;
            }
            if ((recurrence.endDate && new Date(recurrence.endDate).getTime() < currentStartTime.getTime()) ||
                currentStartTime.getTime() === prevStartTime.getTime()) {
                break;
            }
        }
    }
    return occurrences;
};
export const getFocusModesFromOccurrences = (focusModes, startDate, endDate) => {
    const schedules = focusModes.map((focusMode) => focusMode.schedule);
    const occurrences = schedules.map((schedule) => schedule ? calculateOccurrences(schedule, startDate, endDate) : []);
    const activeFocusModes = focusModes.filter((focusMode, index) => occurrences[index].length > 0);
    return activeFocusModes;
};
export const getActiveFocusMode = (currentlyActive, focusModes) => {
    if (currentlyActive) {
        if (currentlyActive.stopCondition === FocusModeStopCondition.Never) {
            return currentlyActive;
        }
        else if (currentlyActive.stopCondition === FocusModeStopCondition.AfterStopTime && currentlyActive.stopTime) {
            if (new Date(currentlyActive.stopTime) > new Date()) {
                return currentlyActive;
            }
        }
    }
    const activeFocusModes = getFocusModesFromOccurrences(focusModes, new Date(), new Date(new Date().getTime() + 1000 * 60));
    if (activeFocusModes.length === 0 && currentlyActive && currentlyActive.stopCondition === FocusModeStopCondition.NextBegins) {
        return currentlyActive;
    }
    if (activeFocusModes.length > 0) {
        return {
            __typename: "ActiveFocusMode",
            mode: activeFocusModes[0],
            stopCondition: FocusModeStopCondition.Automatic,
        };
    }
    return focusModes.length > 0 ? {
        __typename: "ActiveFocusMode",
        mode: focusModes[0],
        stopCondition: FocusModeStopCondition.Automatic,
    } : null;
};
//# sourceMappingURL=focusModes.js.map