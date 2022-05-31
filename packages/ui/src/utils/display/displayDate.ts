/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display
 * @param showDateAndTime Whether to display the time and date, or just the date
 */
export const displayDate = (timestamp: number, showDateAndTime: boolean = true): string => {
    console.log('DISPLAY DATE A', timestamp, showDateAndTime);
    // Create date object
    const date = new Date(timestamp);
    console.log('DISPLAY DATE B', date);
    // Only display year if it's not the current year
    const year = (date.getFullYear() !== new Date().getFullYear()) ? 'numeric' : undefined;
    console.log('DISPLAY DATE C', year);
    // Only display month if it's not the current day or year
    const month = (date.getDate() !== new Date().getDate() || year) ? 'short' : undefined;
    console.log('DISPLAY DATE D', month);
    // Only display day if it's not the current day or year
    const day = (date.getDate() !== new Date().getDate() || year) ? 'numeric' : undefined;
    console.log('DISPLAY DATE E', day);
    // Get date string
    const dateString = (year || month || day) ? date.toLocaleDateString(navigator.language, { year, month, day }) : 'Today at';
    console.log('DISPLAY DATE F', dateString);
    // Get time string
    const timeString = date.toLocaleTimeString(navigator.language);
    console.log('DISPLAY DATE G', timeString);
    // Return date and/or time string
    // If joined today, display time instead of date
    if (dateString === 'Today at') {
        return timeString;
    }
    return showDateAndTime ? `${dateString} ${timeString}` : dateString;
}