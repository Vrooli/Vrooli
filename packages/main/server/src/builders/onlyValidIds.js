import { uuidValidate } from "@local/uuid";
export const onlyValidIds = (ids) => ids.filter(id => typeof id === "string" && uuidValidate(id));
//# sourceMappingURL=onlyValidIds.js.map