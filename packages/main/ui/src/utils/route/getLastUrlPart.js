export const getLastUrlPart = (offset = 0) => {
    let parts = window.location.pathname.split("/");
    parts = parts.filter(part => part !== "");
    if (parts.length < offset + 1)
        return "";
    return parts[parts.length - offset - 1];
};
//# sourceMappingURL=getLastUrlPart.js.map