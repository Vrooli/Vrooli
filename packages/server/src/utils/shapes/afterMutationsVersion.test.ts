/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect, describe, it } from "vitest";
import { findLatestPublicVersionIndex, getChangedVersions, prepareVersionUpdates, sortVersions } from "./afterMutationsVersion.js";

describe("sortVersions", () => {
    it("should correctly sort an array of versions by major, moderate, and minor", () => {
        const versions = [
            { versionLabel: "1.2.3" },
            { versionLabel: "1.2.1" },
            { versionLabel: "2.1.1" },
            { versionLabel: "1.3.1" },
            { versionLabel: "0.9.9" },
        ];
        const sorted = sortVersions(versions);
        expect(sorted.map(v => v.versionLabel)).toEqual(["0.9.9", "1.2.1", "1.2.3", "1.3.1", "2.1.1"]);
    });

    it("should return an empty array when provided with a non-array input", () => {
        const invalidInput = "not an array";
        // @ts-ignore: Testing runtime scenario
        const sorted = sortVersions(invalidInput);
        expect(sorted).toEqual([]);
    });

    it("should return an empty array when provided with an empty array", () => {
        const emptyArray = [];
        const sorted = sortVersions(emptyArray);
        expect(sorted).toEqual([]);
    });

    it("should sort versions that are identical", () => {
        const versions = [
            { versionLabel: "1.1.1" },
            { versionLabel: "1.1.1" },
            { versionLabel: "1.1.1" },
        ];
        const sorted = sortVersions(versions);
        expect(sorted.map(v => v.versionLabel)).toEqual(["1.1.1", "1.1.1", "1.1.1"]);
    });

    it("should maintain the order of elements with the same version", () => {
        // Assuming stable sort
        const versions = [
            { versionLabel: "1.1.1", name: "A" },
            { versionLabel: "1.1.1", name: "B" },
            { versionLabel: "1.1.1", name: "C" },
        ];
        const sorted = sortVersions(versions);
        expect(sorted.map(v => v.name)).toEqual(["A", "B", "C"]);
    });
});

describe("findLatestPublicVersionIndex", () => {
    it("returns -1 if no versions are available", () => {
        expect(findLatestPublicVersionIndex([])).toBe(-1);
    });

    it("returns -1 if all versions are private", () => {
        const versions = [
            { id: "1", isPrivate: true },
            { id: "2", isPrivate: true },
        ];
        // @ts-ignore: Testing runtime scenario
        expect(findLatestPublicVersionIndex(versions)).toBe(-1);
    });

    it("returns the correct index for a single public version", () => {
        const versions = [
            { id: "1", isPrivate: false },
        ];
        // @ts-ignore: Testing runtime scenario
        expect(findLatestPublicVersionIndex(versions)).toBe(0);
    });

    it("returns the index of the last public version when multiple are public", () => {
        const versions = [
            { id: "1", isPrivate: true },
            { id: "2", isPrivate: false },
            { id: "3", isPrivate: false },
        ];
        // @ts-ignore: Testing runtime scenario
        expect(findLatestPublicVersionIndex(versions)).toBe(2);
    });

    it("correctly identifies the public version at the boundaries", () => {
        const versions = [
            { id: "1", isPrivate: false },
            { id: "2", isPrivate: true },
            { id: "3", isPrivate: true },
        ];
        // @ts-ignore: Testing runtime scenario
        expect(findLatestPublicVersionIndex(versions)).toBe(0);
    });

    it("handles large data sets efficiently", () => {
        const versions = Array.from({ length: 10000 }, (_, index) => ({
            id: String(index),
            isPublic: index === 9999, // only the last one is public
        }));
        // @ts-ignore: Testing runtime scenario
        expect(findLatestPublicVersionIndex(versions)).toBe(9999);
    });
});

describe("getChangedVersions", () => {
    it("identifies added, changed, and deleted versions", () => {
        const original = [{ id: BigInt(1), isLatest: true, isLatestPublic: false, versionIndex: 0, isPrivate: false, versionLabel: "1.0" }];
        const updated = [
            { id: BigInt(1), isLatest: true, isLatestPublic: true, versionIndex: 0, isPrivate: false, versionLabel: "1.0" },
            { id: BigInt(2), isLatest: false, isLatestPublic: false, versionIndex: 1, isPrivate: false, versionLabel: "1.1" },
        ];
        const changes = getChangedVersions(original, updated);
        expect(changes).toEqual([
            { id: BigInt(1), isLatest: true, isLatestPublic: true, versionIndex: 0, isPrivate: false, versionLabel: "1.0" },
            { id: BigInt(2), isLatest: false, isLatestPublic: false, versionIndex: 1, isPrivate: false, versionLabel: "1.1" },
        ]);
    });

    it("returns an empty array if there are no changes", () => {
        const original = [{ id: BigInt(1), isLatest: true, isLatestPublic: true, versionIndex: 0, isPrivate: false, versionLabel: "1.0" }];
        const updated = [{ id: BigInt(1), isLatest: true, isLatestPublic: true, versionIndex: 0, isPrivate: false, versionLabel: "1.0" }];
        expect(getChangedVersions(original, updated)).toEqual([]);
    });
});

describe("prepareVersionUpdates", () => {
    it("no versions, so no changes", () => {
        const root = {
            id: "root1",
            versions: [],
        };

        const updatedVersions = prepareVersionUpdates(root);
        expect(updatedVersions.length).toBe(0);
    });

    it("already correct, so no changes", () => {
        const root = {
            id: "root1",
            versions: [
                { id: BigInt(1), isPrivate: false, versionLabel: "1.0", versionIndex: 0, isLatest: false, isLatestPublic: false },
                { id: BigInt(2), isPrivate: false, versionLabel: "1.1", versionIndex: 1, isLatest: false, isLatestPublic: true },
                { id: BigInt(3), isPrivate: true, versionLabel: "1.2", versionIndex: 2, isLatest: true, isLatestPublic: false },
            ],
        };

        const updatedVersions = prepareVersionUpdates(root);
        expect(updatedVersions.length).toBe(0);
    });

    it("index change", () => {
        const root = {
            id: "root1",
            versions: [
                { id: BigInt(1), isPrivate: false, versionLabel: "1.0", versionIndex: 3, isLatest: false, isLatestPublic: false }, // Index incorrect
                { id: BigInt(2), isPrivate: false, versionLabel: "1.1", versionIndex: 4, isLatest: false, isLatestPublic: true }, // Index incorrect
                { id: BigInt(3), isPrivate: true, versionLabel: "1.2", versionIndex: 4, isLatest: true, isLatestPublic: false }, // Index incorrect
            ],
        };

        const updatedVersions = prepareVersionUpdates(root);
        expect(updatedVersions.length).toBe(3);
        expect(updatedVersions.find(v => v.where.id === BigInt(1))).toEqual({ where: { id: BigInt(1) }, data: { versionIndex: 0, isLatest: false, isLatestPublic: false } });
        expect(updatedVersions.find(v => v.where.id === BigInt(2))).toEqual({ where: { id: BigInt(2) }, data: { versionIndex: 1, isLatest: false, isLatestPublic: true } });
        expect(updatedVersions.find(v => v.where.id === BigInt(3))).toEqual({ where: { id: BigInt(3) }, data: { versionIndex: 2, isLatest: true, isLatestPublic: false } });
    });

    it("isLatest change", () => {
        const root = {
            id: "root1",
            versions: [
                { id: BigInt(1), isPrivate: false, versionLabel: "1.0", versionIndex: 0, isLatest: true, isLatestPublic: false }, // Is not latest
                { id: BigInt(2), isPrivate: false, versionLabel: "1.1", versionIndex: 1, isLatest: false, isLatestPublic: true }, // This one's fine, so it should not be updated
                { id: BigInt(3), isPrivate: true, versionLabel: "1.2", versionIndex: 2, isLatest: false, isLatestPublic: false }, // Is latest
            ],
        };

        const updatedVersions = prepareVersionUpdates(root);
        expect(updatedVersions.length).toBe(2);
        expect(updatedVersions.find(v => v.where.id === BigInt(1))).toEqual({ where: { id: BigInt(1) }, data: { versionIndex: 0, isLatest: false, isLatestPublic: false } });
        expect(updatedVersions.find(v => v.where.id === BigInt(3))).toEqual({ where: { id: BigInt(3) }, data: { versionIndex: 2, isLatest: true, isLatestPublic: false } });
    });

    it("isLatestPublic change", () => {
        const root = {
            id: "root1",
            versions: [
                { id: BigInt(1), isPrivate: false, versionLabel: "1.0", versionIndex: 0, isLatest: false, isLatestPublic: false },
                { id: BigInt(2), isPrivate: false, versionLabel: "1.1", versionIndex: 1, isLatest: false, isLatestPublic: false }, // Is latest public
                { id: BigInt(3), isPrivate: true, versionLabel: "1.2", versionIndex: 2, isLatest: true, isLatestPublic: false },
            ],
        };

        const updatedVersions = prepareVersionUpdates(root);
        expect(updatedVersions.length).toBe(1);
        expect(updatedVersions.find(v => v.where.id === BigInt(2))).toEqual({ where: { id: BigInt(2) }, data: { versionIndex: 1, isLatest: false, isLatestPublic: true } });
    });

    it("multiple changes", () => {
        const root = {
            id: "root1",
            versions: [
                { id: BigInt(1), isPrivate: false, versionLabel: "1.0", versionIndex: 0, isLatest: true, isLatestPublic: true },
                { id: BigInt(2), isPrivate: false, versionLabel: "1.1", versionIndex: 0, isLatest: true, isLatestPublic: true },
                { id: BigInt(3), isPrivate: true, versionLabel: "1.2", versionIndex: 0, isLatest: true, isLatestPublic: true },
            ],
        };

        const updatedVersions = prepareVersionUpdates(root);
        expect(updatedVersions.length).toBe(3);
        expect(updatedVersions.find(v => v.where.id === BigInt(1))).toEqual({ where: { id: BigInt(1) }, data: { versionIndex: 0, isLatest: false, isLatestPublic: false } });
        expect(updatedVersions.find(v => v.where.id === BigInt(2))).toEqual({ where: { id: BigInt(2) }, data: { versionIndex: 1, isLatest: false, isLatestPublic: true } });
        expect(updatedVersions.find(v => v.where.id === BigInt(3))).toEqual({ where: { id: BigInt(3) }, data: { versionIndex: 2, isLatest: true, isLatestPublic: false } });
    });
});
