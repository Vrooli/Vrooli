import { getLogic } from "../getters";
export function visibilityBuilder({ objectType, userData, visibility, }) {
    const { validate } = getLogic(["validate"], objectType, userData?.languages ?? ["en"], "visibilityBuilder");
    if (!visibility || visibility === "Public" || !userData) {
        return validate.visibility.public;
    }
    else if (visibility === "Private") {
        return validate.visibility.private;
    }
    else {
        return validate.visibility.owner(userData.id);
    }
}
//# sourceMappingURL=visibilityBuilder.js.map