export const toDatetimeLocal = (date) => {
    const localDate = new Date(date);
    const yyyy = localDate.getFullYear().toString().padStart(4, "0");
    const mm = (localDate.getMonth() + 1).toString().padStart(2, "0");
    const dd = localDate.getDate().toString().padStart(2, "0");
    const hh = localDate.getHours().toString().padStart(2, "0");
    const min = localDate.getMinutes().toString().padStart(2, "0");
    return `${yyyy}-${mm}-${dd}T${hh}:${min}`;
};
export const fromDatetimeLocal = (datetimeLocal) => {
    const [date, time] = datetimeLocal.split("T");
    const [yyyy, mm, dd] = date.split("-");
    const [hh, min] = time.split(":");
    return new Date(parseInt(yyyy), parseInt(mm) - 1, parseInt(dd), parseInt(hh), parseInt(min));
};
//# sourceMappingURL=dateTools.js.map