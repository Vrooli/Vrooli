import { type ResourceVersion } from "../api/types.js";

export enum RelatedResourceLabel {
    /** 
     * Indicates the related ResourceVersion (which should be a "Standard")
     * defines the schema or standard for an INPUT field of the source routine.
     * Convention: Expect a second label like "field:inputName" to specify the field.
     * e.g., labels: [DEFINES_STANDARD_FOR_INPUT_FIELD, "field:age"]
     */
    DEFINES_STANDARD_FOR_INPUT_FIELD = "definesStandardForInputField",
    /** 
     * Indicates the related ResourceVersion (which should be a "Standard")
     * defines the schema or standard for an OUTPUT field of the source routine.
     * Convention: Expect a second label like "field:outputName" to specify the field.
     * e.g., labels: [DEFINES_STANDARD_FOR_OUTPUT_FIELD, "field:reportData"]
     */
    DEFINES_STANDARD_FOR_OUTPUT_FIELD = "definesStandardForOutputField",
    /** Indicates the routine uses the related CodeVersion for its `callDataCode` execution. */
    USES_CODE_VERSION = "usesCodeVersion",
    // Add more as needed
}

/**
 * Represents the structure of a link in the `relatedVersions` array of a ResourceVersion.
 * This links a source ResourceVersion to a target ResourceVersion with a set of labels.
 */
export interface RelatedVersionLink {
    /** The ID of the target ResourceVersion. */
    targetVersionId: string;
    /** 
     * Labels describing the nature of the relationship. 
     * This is a string array that can contain RelatedResourceLabel enum values 
     * as well as other patterned strings (e.g., "field:inputName").
     */
    labels: string[];
    /** 
     * Optional: The actual target ResourceVersion object (e.g., the Standard itself).
     * This might be populated if the data is fetched with includes.
     * Used by SubroutineExecutor to get props and codeLanguage from the Standard.
     */
    targetVersionObject?: Partial<ResourceVersion>;
}

/**
 * Simplified representation of a ResourceVersion for the purpose of these utility functions.
 * Assumes it has an id and a list of related version links.
 */
export interface ResourceVersionLike {
    id: string;
    relatedVersions: RelatedVersionLink[];
}

/**
 * Utility class for managing and interpreting related resource links.
 */
export class RelatedResourceUtils {
    /**
     * Extracts a field identifier (e.g., "inputName") from a link's labels
     * if the link represents a standard for a specific field, using the computed key format.
     * e.g., if link.labels contains "definesStandardForInputField:inputName", this returns "inputName".
     *
     * @param link The RelatedVersionLink to inspect.
     * @param fieldTypeDefiningLabel The primary RelatedResourceLabel that prefixes the field name (e.g., DEFINES_STANDARD_FOR_INPUT_FIELD).
     * @returns The field identifier string if a matching computed key is found, otherwise undefined.
     */
    public static getFieldIdentifierFromLink(link: RelatedVersionLink, fieldTypeDefiningLabel: RelatedResourceLabel): string | undefined {
        const prefix = fieldTypeDefiningLabel + ":";
        for (const labelString of link.labels) {
            if (labelString.startsWith(prefix)) {
                return labelString.substring(prefix.length);
            }
        }
        return undefined;
    }

    /**
     * Adds specified labels to a target resource within a source resource's relatedVersions.
     * If the target resource is not yet related, it adds a new link.
     * If the target resource is already related, it adds the new labels to the existing ones (ensuring uniqueness).
     * Modifies the `relatedVersions` array of the `sourceResource` directly.
     *
     * @param sourceResource The source resource object (must have `id` and `relatedVersions` properties).
     * @param targetVersionId The ID of the target resource to relate.
     * @param labelsToAdd An array of string labels to add to the relationship.
     * @returns The modified `sourceResource` (primarily for chaining, as it's mutated directly).
     */
    public static addRelatedResourceLabels(
        sourceResource: ResourceVersionLike,
        targetVersionId: string,
        labelsToAdd: string[],
    ): ResourceVersionLike {
        if (!sourceResource.relatedVersions) {
            sourceResource.relatedVersions = [];
        }

        const existingLink = sourceResource.relatedVersions.find(r => r.targetVersionId === targetVersionId);

        if (existingLink) {
            labelsToAdd.forEach(label => {
                if (!existingLink.labels.includes(label)) {
                    existingLink.labels.push(label);
                }
            });
        } else {
            const newLabels: string[] = [...new Set(labelsToAdd)];
            const newLink: RelatedVersionLink = {
                targetVersionId,
                labels: newLabels,
            };
            sourceResource.relatedVersions.push(newLink);
        }
        return sourceResource;
    }

    /**
     * Removes specified labels from a target resource within a source resource's relatedVersions.
     * If removing labels results in an empty label list for that target resource, the entire link is removed.
     * Modifies the `relatedVersions` array of the `sourceResource` directly.
     *
     * @param sourceResource The source resource object.
     * @param targetVersionId The ID of the target resource.
     * @param labelsToRemove An array of string labels to remove from the relationship.
     * @returns The modified `sourceResource`.
     */
    public static removeRelatedResourceLabels(
        sourceResource: ResourceVersionLike,
        targetVersionId: string,
        labelsToRemove: string[],
    ): ResourceVersionLike {
        if (!sourceResource.relatedVersions) {
            return sourceResource;
        }

        const linkIndex = sourceResource.relatedVersions.findIndex(r => r.targetVersionId === targetVersionId);

        if (linkIndex > -1) {
            const link = sourceResource.relatedVersions[linkIndex];
            // Both link.labels and labelsToRemove elements are strings.
            link.labels = link.labels.filter(label => !labelsToRemove.includes(label));

            if (link.labels.length === 0) {
                sourceResource.relatedVersions.splice(linkIndex, 1);
            }
        }
        return sourceResource;
    }

    /**
     * Checks if a source resource is related to a target resource with a specific label.
     *
     * @param sourceResource The source resource object.
     * @param targetVersionId The ID of the target resource.
     * @param label The specific string label to check for.
     * @returns True if the relationship with the specified label exists, false otherwise.
     */
    public static hasRelatedResourceLabel(
        sourceResource: ResourceVersionLike | undefined | null,
        targetVersionId: string,
        label: string,
    ): boolean {
        if (!sourceResource || !sourceResource.relatedVersions) {
            return false;
        }
        const link = sourceResource.relatedVersions.find(r => r.targetVersionId === targetVersionId);
        return !!link && link.labels.includes(label);
    }

    /**
     * Gets all target resource IDs that are related to the source resource with a specific label.
     *
     * @param sourceResource The source resource object.
     * @param label The string label to filter by.
     * @returns An array of `targetVersionId`s that have the specified label.
     */
    public static getRelatedResourcesByLabel(
        sourceResource: ResourceVersionLike | undefined | null,
        label: string,
    ): string[] {
        if (!sourceResource || !sourceResource.relatedVersions) {
            return [];
        }
        // link.labels is string[], label is string.
        return sourceResource.relatedVersions
            .filter(link => link.labels.includes(label))
            .map(link => link.targetVersionId);
    }

    /**
     * Gets all RelatedVersionLink objects for a source resource that include a specific label.
     *
     * @param sourceResource The source resource object.
     * @param label The string label to filter by.
     * @returns An array of `RelatedVersionLink` objects that have the specified label.
     */
    public static getRelatedVersionLinksByLabel(
        sourceResource: ResourceVersionLike | undefined | null,
        label: string,
    ): RelatedVersionLink[] {
        if (!sourceResource || !sourceResource.relatedVersions) {
            return [];
        }
        return sourceResource.relatedVersions.filter(link => link.labels.includes(label));
    }
}
