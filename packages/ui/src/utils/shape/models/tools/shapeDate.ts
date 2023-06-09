/**
 * Prepares date string for the database
 */
export const shapeDate = (dateStr: string): string => {
    // Create a new Date object from the local date string
    const date = new Date(dateStr);

    // Check if date is Invalid Date
    if (isNaN(date.getTime())) {
        throw new Error("Invalid date format");
    }

    // Return the date string in the format 'YYYY-MM-DDTHH:MM:SS.SSSZ'
    return date.toISOString();
};
