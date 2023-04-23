export const onlyValidHandles = (handles) => handles.filter(handle => typeof handle === "string" && handle.match(/^\$[a-zA-Z0-9]{3,16}$/));
//# sourceMappingURL=onlyValidHandles.js.map