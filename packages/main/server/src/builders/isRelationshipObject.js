import { isObject } from "@local/utils";
export const isRelationshipObject = (obj) => isObject(obj) && Object.prototype.toString.call(obj) !== "[object Date]";
//# sourceMappingURL=isRelationshipObject.js.map