/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display
 */
export const displayDate = (timestamp: number): string => {
    // Create date object
    const date = new Date(timestamp);
    // Only display year if it's not the current year
    const year = (date.getFullYear() !== new Date().getFullYear()) ? 'numeric' : undefined;
    // Only display month if it's not the current day or year
    const month = (date.getDate() !== new Date().getDate() || year) ? 'short' : undefined;
    // Only display day if it's not the current day or year
    const day = (date.getDate() !== new Date().getDate() || year) ? 'numeric' : undefined;
    // Get date string
    const dateString = (year || month || day) ? date.toLocaleDateString(navigator.language, { year, month, day }) : 'Today at';
    // Get time string
    const timeString = date.toLocaleTimeString(navigator.language);
    // Return date and time string
    return `${dateString} ${timeString}`;
}