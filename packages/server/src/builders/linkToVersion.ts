import { CustomError } from "../events/error";

/**
 * Versioned models shape create and update data from the perspective of the root object. 
 * When using this for relationships, sometimes we want to link to a version instead of
 * the root object. This function will take create or update data shaped for the root object, 
 * and convert it so version is top-level.
 * 
 * NOTE: If the create/update data specifies more than one version, an an error will 
 * be thrown. This is because we don't know which version to link to. For one-to-many 
 * relationships, an array of root objects is expected instead of an array of versions.
 * 
 * @param shaped The shaped data to convert. Either single object or array
 * @param isAdd True if this is a create operation, false if update. This limits 
 * the types of operations allowed
 * @param languages Preferred languages for error messages
 * @returns The shaped data, with version as top-level and the previous top-level under "root"
 */
export const linkToVersion = <T extends { [x: string]: any }>(
    shaped: T | T[],
    isAdd: boolean,
    languages: string[],
) => {
    // If array, recursively call this function for each element
    if (Array.isArray(shaped)) return shaped.map(e => linkToVersion(e, isAdd, languages));
    let version: { [x: string]: any };
    // Now we know it's a single object
    // Make sure there is exactly one version,
    if (!shaped.versions) throw new CustomError("0356", "InvalidArgs", languages);
    if (Array.isArray(shaped.versions)) {
        if (shaped.versions.length !== 1) throw new CustomError("0357", "InvalidArgs", languages);
        version = shaped.versions[0];
    } else {
        version = shaped.versions;
    }
    // Since data is in shape of Prisma, version should contain either 
    // "create" or "update", but not both. Which one is defined depends on
    // whether this is a create or update operation
    if (isAdd) {
        if (!version.create) throw new CustomError("0358", "InvalidArgs", languages);
        if (Array.isArray(version.create)) {
            if (version.create.length !== 1) throw new CustomError("0359", "InvalidArgs", languages);
            // Remove "create" wrapper
            version = version.create[0];
        } else {
            // Remove "create" wrapper
            version = version.create;
        }
    } else {
        if (!version.update) throw new CustomError("0360", "InvalidArgs", languages);
        if (Array.isArray(version.update)) {
            if (version.update.length !== 1) throw new CustomError("0361", "InvalidArgs", languages);
            // Remove "update" wrapper
            version = version.update[0];
        } else {
            // Remove "update" wrapper
            version = version.update;
        }
    }
    // Now we know that version is a single object
    // Remove version from shaped, and add it to the top-level
    delete shaped.versions;
    return {
        ...shaped,
        version,
    };
};
