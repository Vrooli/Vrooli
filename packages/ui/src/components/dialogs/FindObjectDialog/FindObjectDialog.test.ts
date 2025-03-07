/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { findObjectTabParams } from "../../../utils/search/objectToSearch.js";
import { convertRootObjectToVersion, getFilteredTabs } from "./FindObjectDialog.js";

describe("getFilteredTabs function", () => {
    it("returns all tabs when no filters are applied", () => {
        const result = getFilteredTabs(undefined, undefined);
        expect(result).toEqual(findObjectTabParams);
    });

    it("returns limited tabs when limitTo is specified", () => {
        const limitTo = ["Api", "DataConverter"] as const;
        const result = getFilteredTabs(limitTo, undefined);
        expect(result.map(tab => tab.key)).toEqual(expect.arrayContaining(limitTo));
    });

    it("returns only versioned tabs when onlyVersioned is true", () => {
        const result = getFilteredTabs(undefined, true);
        // Expect all object types that can be versioned
        const expectedKeys = [
            "Api",
            "DataConverter",
            "DataStructure",
            "Note",
            "Project",
            "Prompt",
            "Routine",
            "SmartContract",
        ] as const;
        expect(result.map(tab => tab.key)).toEqual(expect.arrayContaining(expectedKeys));
        expect(result.length).to.equal(expectedKeys.length);
    });

    it("returns limited versioned tabs when both limitTo and onlyVersioned are specified", () => {
        const limitTo = ["Api", "Note", "Bot"] as const;
        const result = getFilteredTabs(limitTo, true);
        const expectedKeys = ["Api", "Note"]; // Users are not versioned
        expect(result.map(tab => tab.key)).toEqual(expect.arrayContaining(expectedKeys));
    });

    it("returns all tabs when limitTo is empty", () => {
        const limitTo = [];
        const result = getFilteredTabs(limitTo, undefined);
        expect(result).toEqual(findObjectTabParams);
    });

    it("handles non-existent keys in limitTo", () => {
        const limitTo = ["NonExistent"] as const;
        // @ts-ignore: Testing runtime scenario
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

    it("returns the version object with root when a valid versionId is provided", () => {
        const versionId = "v1";
        const expected = {
            ...v1,
            root: rootWithoutVersions,
        };
        const result = convertRootObjectToVersion(rootObject, versionId);
        expect(result).toEqual(expected);
    });

    it("returns undefined when the versionId does not exist in the versions array", () => {
        const versionId = "v3"; // Non-existent version
        const result = convertRootObjectToVersion(rootObject, versionId);
        expect(result).to.be.undefined;
    });

    it("returns undefined when versionId is undefined", () => {
        // @ts-ignore: Testing runtime scenario
        const result = convertRootObjectToVersion(rootObject, undefined);
        expect(result).to.be.undefined;
    });

    it("correctly handles objects without a versions array", () => {
        const objectWithoutVersions = {
            name: "No Versions Object",
            type: "Original",
        };
        const result = convertRootObjectToVersion(objectWithoutVersions, "v1");
        expect(result).to.be.undefined;
    });

    it("ensures root property does not include versions array", () => {
        const versionId = "v1";
        const result = convertRootObjectToVersion(rootObject, versionId);
        expect(result.root.versions).to.be.undefined;
    });
});
