/**
 * Returns the first non-blank, non-whitespace string
 * @param strings Strings to check
 * @returns First non-blank, non-whitespace string, or empty string if none found
 */
export const firstString = (...strings: (string | null | undefined | (() => string))[]): string => {
    for (const obj of strings) {
        const str = typeof obj === 'function' ? obj() : obj;
        if (str && str.trim() !== '') return str;
    }
    return '';
}

/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display
 * @param showDateAndTime Whether to display the time and date, or just the date
 */
 export const displayDate = (timestamp: number, showDateAndTime: boolean = true): string => {
    // Create date object
    const date = new Date(timestamp);
    // Only display year if it's not the current year
    const year = (date.getFullYear() !== new Date().getFullYear()) ? 'numeric' : undefined;
    // Only display month if it's not the current day or year
    const month = (date.getMonth() !== new Date().getMonth() || year) ? 'short' : undefined;
    // Only display day if it's not the current day or year
    const day = (date.getDate() !== new Date().getDate() || month) ? 'numeric' : undefined;
    // Get date string
    const dateString = (year || month || day) ? date.toLocaleDateString(navigator.language, { year, month, day }) : 'Today at';
    // Get time string
    const timeString = date.toLocaleTimeString(navigator.language);
    // Return date and/or time string
    // If joined today, display time instead of date
    if (dateString === 'Today at') {
        return timeString;
    }
    return showDateAndTime ? `${dateString} ${timeString}` : dateString;
}