import { CustomError } from "../events";
export const linkToVersion = (shaped, isAdd, languages) => {
    if (Array.isArray(shaped))
        return shaped.map(e => linkToVersion(e, isAdd, languages));
    let version;
    if (!shaped.versions)
        throw new CustomError("0356", "InvalidArgs", languages);
    if (Array.isArray(shaped.versions)) {
        if (shaped.versions.length !== 1)
            throw new CustomError("0357", "InvalidArgs", languages);
        version = shaped.versions[0];
    }
    else {
        version = shaped.versions;
    }
    if (isAdd) {
        if (!version.create)
            throw new CustomError("0358", "InvalidArgs", languages);
        if (Array.isArray(version.create)) {
            if (version.create.length !== 1)
                throw new CustomError("0359", "InvalidArgs", languages);
            version = version.create[0];
        }
        else {
            version = version.create;
        }
    }
    else {
        if (!version.update)
            throw new CustomError("0360", "InvalidArgs", languages);
        if (Array.isArray(version.update)) {
            if (version.update.length !== 1)
                throw new CustomError("0361", "InvalidArgs", languages);
            version = version.update[0];
        }
        else {
            version = version.update;
        }
    }
    delete shaped.versions;
    return {
        ...shaped,
        version,
    };
};
//# sourceMappingURL=linkToVersion.js.map