import { isRelationshipObject } from ".";
export const isRelationshipArray = (obj) => Array.isArray(obj) && obj.every(isRelationshipObject);
//# sourceMappingURL=isRelationshipArray.js.map