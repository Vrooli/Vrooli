/**
 * Format a Date object as a string for use with a datetime-local input field.
 *
 * @param {Date} date - The Date object to be formatted.
 * @returns {string} - A formatted string in the "YYYY-MM-DDTHH:mm" format.
 */
export function toDatetimeLocal(date: Date) {
    const localDate = new Date(date);
    const yyyy = localDate.getFullYear().toString().padStart(4, "0");
    const mm = (localDate.getMonth() + 1).toString().padStart(2, "0");
    const dd = localDate.getDate().toString().padStart(2, "0");
    const hh = localDate.getHours().toString().padStart(2, "0");
    const min = localDate.getMinutes().toString().padStart(2, "0");

    return `${yyyy}-${mm}-${dd}T${hh}:${min}`;
}

/**
 * Convert a datetime-local string to a Date object.
 * 
 * @param {string} datetimeLocal - The datetime-local string to be converted.
 * @returns {Date} - A Date object.
 */
export function fromDatetimeLocal(datetimeLocal: string) {
    const [date, time] = datetimeLocal.split("T");
    if (!date || !time) {
        throw new Error("Invalid datetime-local string");
    }
    const [yyyy, mm, dd] = date.split("-");
    if (!yyyy || yyyy.length !== 4 || !mm || mm.length !== 2 || !dd || dd.length !== 2) {
        throw new Error("Invalid date format");
    }
    const [hh, min] = time.split(":");
    if (!hh || hh.length !== 2 || !min || min.length !== 2) {
        throw new Error("Invalid time format");
    }

    return new Date(parseInt(yyyy), parseInt(mm) - 1, parseInt(dd), parseInt(hh), parseInt(min));
}
