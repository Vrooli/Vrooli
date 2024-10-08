/**
 * Prepares date string for the database
 * @param dateStr Date string to shape
 * @param minDate Minimum date allowed
 * @param maxDate Maximum date allowed
 * @returns Shaped date string, or null if date is invalid
 */
export const shapeDate = (
    dateStr: string,
    minDate: Date = new Date("2023-01-01"),
    maxDate: Date = new Date("2100-01-01"),
): string | null => {
    // Create a new Date object from the local date string
    const date = new Date(dateStr);

    // Check if date is Invalid Date
    if (date.toString() === "Invalid Date") {
        return null;
    }

    // Check if date is before minDate
    if (date < minDate) {
        return null;
    }

    // Check if date is after maxDate
    if (date > maxDate) {
        return null;
    }

    // Return the date string in the format 'YYYY-MM-DDTHH:MM:SS.SSSZ'
    return date.toISOString();
};
