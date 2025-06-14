import { describe, it, expect } from "vitest";
import { RelatedResourceLabel, RelatedResourceUtils, type RelatedVersionLink, type ResourceVersionLike } from "./relatedResource.js";

describe("RelatedResourceUtils", () => {
    describe("getFieldIdentifierFromLink", () => {
        it("should extract field identifier from input field label", () => {
            const link: RelatedVersionLink = {
                targetVersionId: "version123",
                labels: ["definesStandardForInputField:username", "someOtherLabel"],
            };
            
            const result = RelatedResourceUtils.getFieldIdentifierFromLink(
                link,
                RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD,
            );
            
            expect(result).toBe("username");
        });

        it("should extract field identifier from output field label", () => {
            const link: RelatedVersionLink = {
                targetVersionId: "version456",
                labels: ["definesStandardForOutputField:reportData", "anotherLabel"],
            };
            
            const result = RelatedResourceUtils.getFieldIdentifierFromLink(
                link,
                RelatedResourceLabel.DEFINES_STANDARD_FOR_OUTPUT_FIELD,
            );
            
            expect(result).toBe("reportData");
        });

        it("should return undefined when no matching label found", () => {
            const link: RelatedVersionLink = {
                targetVersionId: "version789",
                labels: ["usesCodeVersion", "someOtherLabel"],
            };
            
            const result = RelatedResourceUtils.getFieldIdentifierFromLink(
                link,
                RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD,
            );
            
            expect(result).toBeUndefined();
        });

        it("should return undefined for empty labels array", () => {
            const link: RelatedVersionLink = {
                targetVersionId: "version000",
                labels: [],
            };
            
            const result = RelatedResourceUtils.getFieldIdentifierFromLink(
                link,
                RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD,
            );
            
            expect(result).toBeUndefined();
        });

        it("should handle field names with special characters", () => {
            const link: RelatedVersionLink = {
                targetVersionId: "version111",
                labels: ["definesStandardForInputField:user_email_address", "other"],
            };
            
            const result = RelatedResourceUtils.getFieldIdentifierFromLink(
                link,
                RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD,
            );
            
            expect(result).toBe("user_email_address");
        });

        it("should handle field names with colons", () => {
            const link: RelatedVersionLink = {
                targetVersionId: "version222",
                labels: ["definesStandardForInputField:namespace:fieldName", "other"],
            };
            
            const result = RelatedResourceUtils.getFieldIdentifierFromLink(
                link,
                RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD,
            );
            
            expect(result).toBe("namespace:fieldName");
        });
    });

    describe("addRelatedResourceLabels", () => {
        it("should add new relationship with labels", () => {
            const source: ResourceVersionLike = {
                id: "source1",
                relatedVersions: [],
            };
            
            const result = RelatedResourceUtils.addRelatedResourceLabels(
                source,
                "target1",
                ["label1", "label2"],
            );
            
            expect(result.relatedVersions).toHaveLength(1);
            expect(result.relatedVersions[0]).toEqual({
                targetVersionId: "target1",
                labels: ["label1", "label2"],
            });
        });

        it("should add labels to existing relationship", () => {
            const source: ResourceVersionLike = {
                id: "source2",
                relatedVersions: [{
                    targetVersionId: "target2",
                    labels: ["existingLabel"],
                }],
            };
            
            const result = RelatedResourceUtils.addRelatedResourceLabels(
                source,
                "target2",
                ["newLabel1", "newLabel2"],
            );
            
            expect(result.relatedVersions).toHaveLength(1);
            expect(result.relatedVersions[0].labels).toEqual([
                "existingLabel",
                "newLabel1",
                "newLabel2",
            ]);
        });

        it("should not add duplicate labels", () => {
            const source: ResourceVersionLike = {
                id: "source3",
                relatedVersions: [{
                    targetVersionId: "target3",
                    labels: ["label1", "label2"],
                }],
            };
            
            const result = RelatedResourceUtils.addRelatedResourceLabels(
                source,
                "target3",
                ["label2", "label3"],
            );
            
            expect(result.relatedVersions[0].labels).toEqual([
                "label1",
                "label2",
                "label3",
            ]);
        });

        it("should handle source without relatedVersions array", () => {
            const source: ResourceVersionLike = {
                id: "source4",
                relatedVersions: undefined as any,
            };
            
            const result = RelatedResourceUtils.addRelatedResourceLabels(
                source,
                "target4",
                ["label1"],
            );
            
            expect(result.relatedVersions).toHaveLength(1);
            expect(result.relatedVersions[0]).toEqual({
                targetVersionId: "target4",
                labels: ["label1"],
            });
        });

        it("should deduplicate labels in new relationship", () => {
            const source: ResourceVersionLike = {
                id: "source5",
                relatedVersions: [],
            };
            
            const result = RelatedResourceUtils.addRelatedResourceLabels(
                source,
                "target5",
                ["label1", "label1", "label2", "label2"],
            );
            
            expect(result.relatedVersions[0].labels).toEqual(["label1", "label2"]);
        });

        it("should preserve targetVersionObject if it exists", () => {
            const source: ResourceVersionLike = {
                id: "source6",
                relatedVersions: [{
                    targetVersionId: "target6",
                    labels: ["existingLabel"],
                    targetVersionObject: { id: "target6", resourceVersionType: "Standard" } as any,
                }],
            };
            
            const result = RelatedResourceUtils.addRelatedResourceLabels(
                source,
                "target6",
                ["newLabel"],
            );
            
            expect(result.relatedVersions[0].targetVersionObject).toBeDefined();
            expect(result.relatedVersions[0].targetVersionObject!.id).toBe("target6");
        });
    });

    describe("removeRelatedResourceLabels", () => {
        it("should remove specified labels from relationship", () => {
            const source: ResourceVersionLike = {
                id: "source1",
                relatedVersions: [{
                    targetVersionId: "target1",
                    labels: ["label1", "label2", "label3"],
                }],
            };
            
            const result = RelatedResourceUtils.removeRelatedResourceLabels(
                source,
                "target1",
                ["label2"],
            );
            
            expect(result.relatedVersions[0].labels).toEqual(["label1", "label3"]);
        });

        it("should remove entire relationship when all labels removed", () => {
            const source: ResourceVersionLike = {
                id: "source2",
                relatedVersions: [{
                    targetVersionId: "target2",
                    labels: ["label1", "label2"],
                }],
            };
            
            const result = RelatedResourceUtils.removeRelatedResourceLabels(
                source,
                "target2",
                ["label1", "label2"],
            );
            
            expect(result.relatedVersions).toHaveLength(0);
        });

        it("should handle non-existent target", () => {
            const source: ResourceVersionLike = {
                id: "source3",
                relatedVersions: [{
                    targetVersionId: "target3",
                    labels: ["label1"],
                }],
            };
            
            const result = RelatedResourceUtils.removeRelatedResourceLabels(
                source,
                "nonExistentTarget",
                ["label1"],
            );
            
            expect(result.relatedVersions).toHaveLength(1);
            expect(result.relatedVersions[0].targetVersionId).toBe("target3");
        });

        it("should handle source without relatedVersions", () => {
            const source: ResourceVersionLike = {
                id: "source4",
                relatedVersions: undefined as any,
            };
            
            const result = RelatedResourceUtils.removeRelatedResourceLabels(
                source,
                "target4",
                ["label1"],
            );
            
            expect(result.relatedVersions).toBeUndefined();
        });

        it("should handle removing non-existent labels", () => {
            const source: ResourceVersionLike = {
                id: "source5",
                relatedVersions: [{
                    targetVersionId: "target5",
                    labels: ["label1", "label2"],
                }],
            };
            
            const result = RelatedResourceUtils.removeRelatedResourceLabels(
                source,
                "target5",
                ["label3", "label4"],
            );
            
            expect(result.relatedVersions[0].labels).toEqual(["label1", "label2"]);
        });

        it("should handle multiple relationships correctly", () => {
            const source: ResourceVersionLike = {
                id: "source6",
                relatedVersions: [
                    {
                        targetVersionId: "target6a",
                        labels: ["label1", "label2"],
                    },
                    {
                        targetVersionId: "target6b",
                        labels: ["label3", "label4"],
                    },
                ],
            };
            
            const result = RelatedResourceUtils.removeRelatedResourceLabels(
                source,
                "target6a",
                ["label1"],
            );
            
            expect(result.relatedVersions).toHaveLength(2);
            expect(result.relatedVersions[0].labels).toEqual(["label2"]);
            expect(result.relatedVersions[1].labels).toEqual(["label3", "label4"]);
        });
    });

    describe("hasRelatedResourceLabel", () => {
        it("should return true when relationship with label exists", () => {
            const source: ResourceVersionLike = {
                id: "source1",
                relatedVersions: [{
                    targetVersionId: "target1",
                    labels: ["label1", "label2"],
                }],
            };
            
            const result = RelatedResourceUtils.hasRelatedResourceLabel(
                source,
                "target1",
                "label2",
            );
            
            expect(result).toBe(true);
        });

        it("should return false when relationship exists but label doesn't", () => {
            const source: ResourceVersionLike = {
                id: "source2",
                relatedVersions: [{
                    targetVersionId: "target2",
                    labels: ["label1", "label2"],
                }],
            };
            
            const result = RelatedResourceUtils.hasRelatedResourceLabel(
                source,
                "target2",
                "label3",
            );
            
            expect(result).toBe(false);
        });

        it("should return false when target doesn't exist", () => {
            const source: ResourceVersionLike = {
                id: "source3",
                relatedVersions: [{
                    targetVersionId: "target3",
                    labels: ["label1"],
                }],
            };
            
            const result = RelatedResourceUtils.hasRelatedResourceLabel(
                source,
                "nonExistentTarget",
                "label1",
            );
            
            expect(result).toBe(false);
        });

        it("should handle null source", () => {
            const result = RelatedResourceUtils.hasRelatedResourceLabel(
                null,
                "target",
                "label",
            );
            
            expect(result).toBe(false);
        });

        it("should handle undefined source", () => {
            const result = RelatedResourceUtils.hasRelatedResourceLabel(
                undefined,
                "target",
                "label",
            );
            
            expect(result).toBe(false);
        });

        it("should handle source without relatedVersions", () => {
            const source: ResourceVersionLike = {
                id: "source6",
                relatedVersions: undefined as any,
            };
            
            const result = RelatedResourceUtils.hasRelatedResourceLabel(
                source,
                "target",
                "label",
            );
            
            expect(result).toBe(false);
        });
    });

    describe("getRelatedResourcesByLabel", () => {
        it("should return target IDs with specified label", () => {
            const source: ResourceVersionLike = {
                id: "source1",
                relatedVersions: [
                    {
                        targetVersionId: "target1",
                        labels: ["label1", "commonLabel"],
                    },
                    {
                        targetVersionId: "target2",
                        labels: ["label2"],
                    },
                    {
                        targetVersionId: "target3",
                        labels: ["commonLabel", "label3"],
                    },
                ],
            };
            
            const result = RelatedResourceUtils.getRelatedResourcesByLabel(
                source,
                "commonLabel",
            );
            
            expect(result).toEqual(["target1", "target3"]);
        });

        it("should return empty array when no matches", () => {
            const source: ResourceVersionLike = {
                id: "source2",
                relatedVersions: [
                    {
                        targetVersionId: "target1",
                        labels: ["label1"],
                    },
                ],
            };
            
            const result = RelatedResourceUtils.getRelatedResourcesByLabel(
                source,
                "nonExistentLabel",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle null source", () => {
            const result = RelatedResourceUtils.getRelatedResourcesByLabel(
                null,
                "label",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle undefined source", () => {
            const result = RelatedResourceUtils.getRelatedResourcesByLabel(
                undefined,
                "label",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle source without relatedVersions", () => {
            const source: ResourceVersionLike = {
                id: "source5",
                relatedVersions: undefined as any,
            };
            
            const result = RelatedResourceUtils.getRelatedResourcesByLabel(
                source,
                "label",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle empty relatedVersions array", () => {
            const source: ResourceVersionLike = {
                id: "source6",
                relatedVersions: [],
            };
            
            const result = RelatedResourceUtils.getRelatedResourcesByLabel(
                source,
                "label",
            );
            
            expect(result).toEqual([]);
        });
    });

    describe("getRelatedVersionLinksByLabel", () => {
        it("should return full link objects with specified label", () => {
            const link1: RelatedVersionLink = {
                targetVersionId: "target1",
                labels: ["label1", "commonLabel"],
                targetVersionObject: { id: "target1" } as any,
            };
            const link2: RelatedVersionLink = {
                targetVersionId: "target2",
                labels: ["label2"],
            };
            const link3: RelatedVersionLink = {
                targetVersionId: "target3",
                labels: ["commonLabel", "label3"],
            };
            
            const source: ResourceVersionLike = {
                id: "source1",
                relatedVersions: [link1, link2, link3],
            };
            
            const result = RelatedResourceUtils.getRelatedVersionLinksByLabel(
                source,
                "commonLabel",
            );
            
            expect(result).toHaveLength(2);
            expect(result[0]).toBe(link1);
            expect(result[1]).toBe(link3);
        });

        it("should return empty array when no matches", () => {
            const source: ResourceVersionLike = {
                id: "source2",
                relatedVersions: [
                    {
                        targetVersionId: "target1",
                        labels: ["label1"],
                    },
                ],
            };
            
            const result = RelatedResourceUtils.getRelatedVersionLinksByLabel(
                source,
                "nonExistentLabel",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle null source", () => {
            const result = RelatedResourceUtils.getRelatedVersionLinksByLabel(
                null,
                "label",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle undefined source", () => {
            const result = RelatedResourceUtils.getRelatedVersionLinksByLabel(
                undefined,
                "label",
            );
            
            expect(result).toEqual([]);
        });

        it("should handle source without relatedVersions", () => {
            const source: ResourceVersionLike = {
                id: "source5",
                relatedVersions: undefined as any,
            };
            
            const result = RelatedResourceUtils.getRelatedVersionLinksByLabel(
                source,
                "label",
            );
            
            expect(result).toEqual([]);
        });

        it("should work with enum values", () => {
            const source: ResourceVersionLike = {
                id: "source6",
                relatedVersions: [
                    {
                        targetVersionId: "target1",
                        labels: [RelatedResourceLabel.USES_CODE_VERSION],
                    },
                    {
                        targetVersionId: "target2",
                        labels: [RelatedResourceLabel.DEFINES_STANDARD_FOR_INPUT_FIELD],
                    },
                ],
            };
            
            const result = RelatedResourceUtils.getRelatedVersionLinksByLabel(
                source,
                RelatedResourceLabel.USES_CODE_VERSION,
            );
            
            expect(result).toHaveLength(1);
            expect(result[0].targetVersionId).toBe("target1");
        });
    });
});
