/* eslint-disable @typescript-eslint/ban-ts-comment */
import { SearchPageTabOption, findObjectTabParams } from "@local/shared";
import { convertRootObjectToVersion, getFilteredTabs } from "./FindObjectDialog";

describe("getFilteredTabs function", () => {
    test("returns all tabs when no filters are applied", () => {
        const result = getFilteredTabs(undefined, undefined);
        expect(result).toEqual(findObjectTabParams);
    });

    test("returns limited tabs when limitTo is specified", () => {
        const limitTo = [SearchPageTabOption.Api, SearchPageTabOption.Code];
        const result = getFilteredTabs(limitTo, undefined);
        expect(result.map(tab => tab.key)).toEqual(expect.arrayContaining(limitTo));
    });

    test("returns only versioned tabs when onlyVersioned is true", () => {
        const result = getFilteredTabs(undefined, true);
        // Expect all object types that can be versioned
        const expectedKeys = [
            SearchPageTabOption.Api,
            SearchPageTabOption.Code,
            SearchPageTabOption.Note,
            SearchPageTabOption.Project,
            SearchPageTabOption.Routine,
            SearchPageTabOption.Standard,
        ];
        expect(result.map(tab => tab.key)).toEqual(expect.arrayContaining(expectedKeys));
    });

    test("returns limited versioned tabs when both limitTo and onlyVersioned are specified", () => {
        const limitTo = [SearchPageTabOption.Api, SearchPageTabOption.Note, SearchPageTabOption.User];
        const result = getFilteredTabs(limitTo, true);
        const expectedKeys = [SearchPageTabOption.Api, SearchPageTabOption.Note]; // Users are not versioned
        expect(result.map(tab => tab.key)).toEqual(expect.arrayContaining(expectedKeys));
    });

    test("returns all tabs when limitTo is empty", () => {
        const limitTo = [];
        const result = getFilteredTabs(limitTo, undefined);
        expect(result).toEqual(findObjectTabParams);
    });

    test("handles non-existent keys in limitTo", () => {
        const limitTo = ["NonExistent"];
        const result = getFilteredTabs(limitTo, undefined);
        expect(result).toEqual([]);
    });
});

describe("convertRootObjectToVersion", () => {
    const rootObject = {
        name: "Root Object",
        type: "Original",
        versions: [
            { id: "v1", name: "Version 1", additionalData: "Data for V1" },
            { id: "v2", name: "Version 2", additionalData: "Data for V2" },
        ],
    };
    const { versions, ...rootWithoutVersions } = rootObject;
    const v1 = rootObject.versions.find(v => v.id === "v1")!;
    const v2 = rootObject.versions.find(v => v.id === "v2")!;

    test("returns the version object with root when a valid versionId is provided", () => {
        const versionId = "v1";
        const expected = {
            ...v1,
            root: rootWithoutVersions,
        };
        const result = convertRootObjectToVersion(rootObject, versionId);
        expect(result).toEqual(expected);
    });

    test("returns undefined when the versionId does not exist in the versions array", () => {
        const versionId = "v3"; // Non-existent version
        const result = convertRootObjectToVersion(rootObject, versionId);
        expect(result).toBeUndefined();
    });

    test("returns undefined when versionId is undefined", () => {
        // @ts-ignore: Testing runtime scenario
        const result = convertRootObjectToVersion(rootObject, undefined);
        expect(result).toBeUndefined();
    });

    test("correctly handles objects without a versions array", () => {
        const objectWithoutVersions = {
            name: "No Versions Object",
            type: "Original",
        };
        const result = convertRootObjectToVersion(objectWithoutVersions, "v1");
        expect(result).toBeUndefined();
    });

    test("ensures root property does not include versions array", () => {
        const versionId = "v1";
        const result = convertRootObjectToVersion(rootObject, versionId);
        expect(result.root.versions).toBeUndefined();
    });
});
