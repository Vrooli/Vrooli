
/**
 * Returns the first non-blank, non-whitespace string
 * @param strings Strings to check
 * @returns First non-blank, non-whitespace string, or empty string if none found
 */
export function firstString(...strings: unknown[]): string {
    for (const obj of strings) {
        let str: string | undefined;

        if (typeof obj === "string") {
            str = obj;
        } else if (typeof obj === "function") {
            const result = obj();
            if (typeof result === "string") {
                str = result;
            }
        }

        if (str && str.trim() !== "") return str;
    }

    return "";
}

/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display as seconds since epoch, or an ISO string
 * @param showDateAndTime Whether to display the time and date, or just the date
 */
export function displayDate(timestamp: number | string, showDateAndTime = true): string | null {
    let date: Date;

    if (typeof timestamp === "number") {
        if (isNaN(timestamp) || timestamp < 0) {
            return null;
        }
        date = new Date(timestamp);
    } else if (typeof timestamp === "string") {
        date = new Date(timestamp);
        if (isNaN(date.getTime())) {
            return null;
        }
    } else {
        return null;
    }

    const currentDate = new Date();

    const isCurrentYear = date.getUTCFullYear() === currentDate.getUTCFullYear();
    const isCurrentMonth = date.getUTCMonth() === currentDate.getUTCMonth();
    const isCurrentDay = date.getUTCDate() === currentDate.getUTCDate();
    const isToday = isCurrentYear && isCurrentMonth && isCurrentDay;

    const year = !isCurrentYear ? "numeric" : undefined;
    const month = !isToday ? "short" : undefined;
    const day = !isToday ? "numeric" : undefined;

    const dateString = !isToday ? date.toLocaleDateString(navigator.language, { year, month, day }) : "Today at";
    const timeString = date.toLocaleTimeString(navigator.language, { timeZone: "UTC" });
    // If joined today, display time instead of date
    if (isToday) {
        return timeString;
    }
    // Return date and/or time string
    return showDateAndTime ? `${dateString}${!isToday ? "," : ""} ${timeString}` : dateString;
}

/**
 * Converts font sizes (rem, em, px) to pixels
 * @param size - font size as string or number
 * @param id - id of element, used to get em size
 * @returns font size number in pixels
 */
export function fontSizeToPixels(size: string | number, id?: string): number {
    if (typeof size === "number") {
        return size;
    }
    if (size.includes("px")) {
        return parseFloat(size);
    }
    if (size.includes("rem")) {
        return parseFloat(size) * 16;
    }
    if (size.includes("em")) {
        if (!id) {
            console.error("Must provide id to convert em to px");
            return 0;
        }
        const el = document.getElementById(id);
        if (el) {
            const fontSize = window.getComputedStyle(el).fontSize;
            return parseFloat(size) * parseFloat(fontSize);
        }
    }
    return 0;
}

const MAX_CONTENT_LENGTH = 1500;

/**
 * Generates a context string from a selection of text, or most/all of the text if no selection.
 * @param selected The current selection
 * @param fullText The full text from the input field
 * @returns The formatted context string
 */
export function generateContext(selected: string, fullText: string): string {
    let context = selected.trim();
    const ellipsis = "…";
    // If selected text is too long, provide the last 1500 characters
    if (context.length > MAX_CONTENT_LENGTH) {
        context = ellipsis + context.substring(context.length - MAX_CONTENT_LENGTH - ellipsis.length, context.length);
        return context.trim();
    } else if (context.length > 0) {
        return context.trim();
    }
    const fullValue = fullText.trim();
    if (fullValue.length <= 0) {
        return "";
    }
    // If there's no selection, provide the full text if it's not too long
    if (fullValue.length <= MAX_CONTENT_LENGTH) context = fullValue;
    // Otherwise, provide the last 1500 characters
    else context = ellipsis + fullValue.substring(fullValue.length - MAX_CONTENT_LENGTH - ellipsis.length, fullValue.length);
    return context.trim();
}

/**
 * Formats an event's time range based on how far in the future it is
 * @param start Start time of the event
 * @param end End time of the event
 * @returns Formatted time string
 */
export function formatEventTime(start: Date, end: Date): string {
    const now = new Date();
    const tomorrow = new Date(now);
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(0, 0, 0, 0);

    const nextWeek = new Date(now);
    nextWeek.setDate(now.getDate() + 7);
    nextWeek.setHours(0, 0, 0, 0);

    // Format time portion (HH:MM)
    function formatTime(date: Date) {
        return date.toLocaleTimeString(navigator.language, {
            hour: "numeric",
            minute: "2-digit",
            hour12: true,
        }).replace(/\s?[AP]M/i, "");
    }

    const timeRange = `${formatTime(start)}-${formatTime(end)}`;

    if (start < tomorrow) {
        // Within 24 hours - just show time
        return timeRange;
    } else if (start < nextWeek) {
        // Within a week - show day and time
        const day = start.toLocaleDateString(navigator.language, { weekday: "short" });
        return `${day} ${timeRange}`;
    } else {
        // Beyond a week - show date and time
        const date = start.toLocaleDateString(navigator.language, {
            month: "numeric",
            day: "numeric",
        });
        return `${date} ${timeRange}`;
    }
}
