import { getLogic } from "../getters";
export const oneIsPublic = (permissionsData, list, languages) => {
    for (let i = 0; i < list.length; i++) {
        const [field, type] = list[i];
        const { validate } = getLogic(["validate"], type, languages, "oneIsPublic");
        if (permissionsData[field] && validate.isPublic(permissionsData[field], languages)) {
            return true;
        }
    }
    return false;
};
//# sourceMappingURL=oneIsPublic.js.map