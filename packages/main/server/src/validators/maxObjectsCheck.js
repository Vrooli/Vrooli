import { CustomError } from "../events";
import { getLogic } from "../getters";
const checkObjectLimitNumber = (count, limit, languages) => {
    if (count > limit) {
        throw new CustomError("0352", "MaxObjectsReached", languages);
    }
};
const checkObjectLimitPremium = (count, hasPremium, limit, languages) => {
    if (hasPremium)
        checkObjectLimitNumber(count, limit.premium, languages);
    else
        checkObjectLimitNumber(count, limit.noPremium, languages);
};
const checkObjectLimitPrivacy = (count, hasPremium, isPrivate, limit, languages) => {
    if (isPrivate) {
        if (typeof limit.private === "number")
            checkObjectLimitNumber(count, limit.private, languages);
        else
            checkObjectLimitPremium(count, hasPremium, limit.private, languages);
    }
    else {
        if (typeof limit.public === "number")
            checkObjectLimitNumber(count, limit.public, languages);
        else
            checkObjectLimitPremium(count, hasPremium, limit.public, languages);
    }
};
const checkObjectLimitOwner = (count, ownerType, hasPremium, isPrivate, limit, languages) => {
    if (ownerType === "User") {
        if (typeof limit.User === "number")
            checkObjectLimitNumber(count, limit.User, languages);
        else if (typeof limit.User.premium !== undefined)
            checkObjectLimitPremium(count, hasPremium, limit.User, languages);
        else
            checkObjectLimitPrivacy(count, hasPremium, isPrivate, limit.User, languages);
    }
    else {
        if (typeof limit.Organization === "number")
            checkObjectLimitNumber(count, limit.Organization, languages);
        else if (typeof limit.Organization.premium !== undefined)
            checkObjectLimitPremium(count, hasPremium, limit.Organization, languages);
        else
            checkObjectLimitPrivacy(count, hasPremium, isPrivate, limit.Organization, languages);
    }
};
const checkObjectLimit = (count, ownerType, hasPremium, isPrivate, limit, languages) => {
    if (typeof limit === "number")
        checkObjectLimitNumber(count, limit, languages);
    else if (typeof limit.premium !== undefined)
        checkObjectLimitPremium(count, hasPremium, limit, languages);
    else if (typeof limit.private !== undefined)
        checkObjectLimitPrivacy(count, hasPremium, isPrivate, limit, languages);
    else
        checkObjectLimitOwner(count, ownerType, hasPremium, isPrivate, limit, languages);
};
export async function maxObjectsCheck(authDataById, idsByAction, prisma, userData) {
    const counts = {};
    if (idsByAction.Create) {
        for (const id of idsByAction.Create) {
            const authData = authDataById[id];
            const { validate } = getLogic(["validate"], authData.__typename, userData.languages, "maxObjectsCheck-create");
            const owners = validate.owner(authData, userData.id);
            const ownerId = owners.Organization?.id ?? owners.User?.id;
            if (!ownerId)
                throw new CustomError("0310", "InternalError", userData.languages);
            counts[authData.__typename] = counts[authData.__typename] || {};
            counts[authData.__typename][ownerId] = counts[authData.__typename][ownerId] || { private: 0, public: 0 };
            const isPublic = validate.isPublic(authData, userData.languages);
            counts[authData.__typename][ownerId][isPublic ? "public" : "private"]++;
        }
    }
    if (idsByAction.Delete) {
        for (const id of idsByAction.Delete) {
            const authData = authDataById[id];
            const { validate } = getLogic(["validate"], authData.__typename, userData.languages, "maxObjectsCheck-delete");
            const owners = validate.owner(authData, userData.id);
            const ownerId = owners.Organization?.id ?? owners.User?.id;
            if (!ownerId)
                throw new CustomError("0311", "InternalError", userData.languages);
            counts[authData.__typename] = counts[authData.__typename] || {};
            counts[authData.__typename][ownerId] = counts[authData.__typename][ownerId] || { private: 0, public: 0 };
            const isPublic = validate.isPublic(authData, userData.languages);
            counts[authData.__typename][ownerId][isPublic ? "public" : "private"]--;
        }
    }
    for (const objectType of Object.keys(counts)) {
        const { delegate, validate } = getLogic(["delegate", "validate"], objectType, userData.languages, "maxObjectsCheck-existing");
        for (const ownerId in counts[objectType]) {
            let currCountPrivate = await delegate(prisma).count({ where: validate.visibility.private });
            let currCountPublic = await delegate(prisma).count({ where: validate.visibility.public });
            currCountPrivate += counts[objectType][ownerId].private;
            currCountPublic += counts[objectType][ownerId].public;
            const maxObjects = validate.maxObjects;
            const ownerType = userData.id === ownerId ? "User" : "Organization";
            const hasPremium = userData.hasPremium;
            checkObjectLimit(currCountPrivate, ownerType, hasPremium, true, maxObjects, userData.languages);
            checkObjectLimit(currCountPublic, ownerType, hasPremium, false, maxObjects, userData.languages);
        }
    }
}
//# sourceMappingURL=maxObjectsCheck.js.map