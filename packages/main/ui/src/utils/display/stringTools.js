export const firstString = (...strings) => {
    for (const obj of strings) {
        const str = typeof obj === "function" ? obj() : obj;
        if (str && str.trim() !== "")
            return str;
    }
    return "";
};
export const displayDate = (timestamp, showDateAndTime = true) => {
    const date = new Date(timestamp);
    const year = (date.getFullYear() !== new Date().getFullYear()) ? "numeric" : undefined;
    const month = "short";
    const day = (date.getDate() !== new Date().getDate() || month) ? "numeric" : undefined;
    const dateString = (year || month || day) ? date.toLocaleDateString(navigator.language, { year, month, day }) : "Today at";
    const timeString = date.toLocaleTimeString(navigator.language);
    if (dateString === "Today at") {
        return timeString;
    }
    return showDateAndTime ? `${dateString} ${timeString}` : dateString;
};
//# sourceMappingURL=stringTools.js.map