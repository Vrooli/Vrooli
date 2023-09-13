/**
 * Returns the first non-blank, non-whitespace string
 * @param strings Strings to check
 * @returns First non-blank, non-whitespace string, or empty string if none found
 */
export const firstString = (...strings: unknown[]): string => {
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
};

/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display
 * @param showDateAndTime Whether to display the time and date, or just the date
 */
export const displayDate = (timestamp: number, showDateAndTime = true): string => {
    // Create date object
    const date = new Date(timestamp);
    // Only display year if it's not the current year
    const year = (date.getFullYear() !== new Date().getFullYear()) ? "numeric" : undefined;
    // Always display month
    const month = "short";
    // Only display day if it's not the current day or year
    const day = (date.getDate() !== new Date().getDate() || month) ? "numeric" : undefined;
    // Get date string
    const dateString = (year || month || day) ? date.toLocaleDateString(navigator.language, { year, month, day }) : "Today at";
    // Get time string
    const timeString = date.toLocaleTimeString(navigator.language);
    // Return date and/or time string
    // If joined today, display time instead of date
    if (dateString === "Today at") {
        return timeString;
    }
    return showDateAndTime ? `${dateString} ${timeString}` : dateString;
};

/**
 * Converts font sizes (rem, em, px) to pixels
 * @param size - font size as string or number
 * @param id - id of element, used to get em size
 * @returns font size number in pixels
 */
export const fontSizeToPixels = (size: string | number, id?: string): number => {
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
};
