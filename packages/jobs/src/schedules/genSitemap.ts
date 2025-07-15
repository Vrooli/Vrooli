// AI_CHECK: TASK_ID=TYPE_SAFETY COUNT=2 | LAST: 2025-07-04
import { DbProvider, ModelMap, UI_URL_REMOTE, logger, type PrismaDelegate } from "@vrooli/server";
import { LINKS, ResourceSubType, ResourceType, generateSitemap, generateSitemapIndex, type SitemapEntryContent } from "@vrooli/shared";
import { 
    MAX_ENTRIES_PER_SITEMAP, 
    MAX_SITEMAP_FILE_SIZE_BYTES, 
    ESTIMATED_ENTRY_OVERHEAD_BYTES, 
    BATCH_SIZE_SMALL, 
} from "../constants.js";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import zlib from "zlib";

const sitemapObjectTypes = [
    "ResourceVersion",
    "Team",
    "User",
] as const;

/**
 * Type guard to check if a value is a PrismaDelegate
 */
function isPrismaDelegate(value: unknown): value is PrismaDelegate {
    return value !== null && 
           typeof value === "object" && 
           "findMany" in value &&
           typeof value.findMany === "function";
}

/**
 * Type guard to check if a string is a valid LINKS key
 */
function isValidLinkKey(key: string): key is keyof typeof LINKS {
    return key in LINKS;
}

/**
 * Type guard to safely check if an object has a specific property
 */
function hasProperty<T extends string>(obj: unknown, prop: T): obj is Record<T, unknown> {
    return obj !== null && typeof obj === "object" && prop in obj;
}

/**
 * Type guard to check if a string is a valid ResourceSubType
 */
function isValidResourceSubType(value: string): value is ResourceSubType {
    return (Object.values(ResourceSubType) as string[]).includes(value);
}

/**
 * Type guard to check if an object is a valid SitemapProperties root object
 */
function isValidRootObject(obj: unknown): obj is SitemapProperties["root"] {
    if (!obj || typeof obj !== "object") return false;
    
    const root = obj as { [key: string]: unknown };
    return typeof root.id === "string" && 
           typeof root.publicId === "string" &&
           (root.handle === undefined || typeof root.handle === "string") &&
           (root.isDeleted === undefined || typeof root.isDeleted === "boolean") &&
           (root.isPrivate === undefined || typeof root.isPrivate === "boolean") &&
           (root.resourceType === undefined || (Object.values(ResourceType) as string[]).includes(root.resourceType as string));
}

/**
 * Defines the structure of properties relevant for sitemap generation, 
 * fetched from the database for each sitemap object type.
 */
type SitemapProperties = {
    /** The internal ID of the object. */
    id: string;
    /** The public-facing ID of the object. */
    publicId: string;
    /** Optional user-defined handle for the object (e.g., for Teams, Users). */
    handle?: string;
    /** Flag indicating if the object is deleted. */
    isDeleted?: boolean;
    /** Specific subtype if the object is a ResourceVersion (e.g., CodeDataConverter, StandardPrompt). */
    resourceSubType?: ResourceSubType;
    /** Optional version label if the object is a ResourceVersion. */
    versionLabel?: string;
    /** A list of available language translations for the object. */
    translations: { language: string }[];
    /** 
     * Optional root object properties, primarily for ResourceVersion objects 
     * to link back to their parent resource.
     */
    root?: {
        /** The internal ID of the root object. */
        id: string;
        /** The public-facing ID of the root object. */
        publicId: string;
        /** Optional user-defined handle for the root object. */
        handle?: string;
        /** Flag indicating if the root object is deleted. */
        isDeleted?: boolean;
        /** Flag indicating if the root object is private. */
        isPrivate?: boolean;
        /** The type of the root resource (e.g., Api, Note, Project). */
        resourceType?: ResourceType;
    };
};

/**
 * Maps sitemap object types to their corresponding base URL paths.
 * This function determines the initial segment of the URL for an object in the sitemap.
 * For `ResourceVersion` objects, it further disambiguates the URL based on `resourceSubType` or `root.resourceType`.
 * 
 * IMPORTANT: The URL structure must align with the og-worker's expectations for generating 
 * Open Graph meta tags. The og-worker relies on these specific URL patterns to reliably 
 * fetch metadata without requiring additional dependencies. URLs include:
 * - /u/@handle for users (the @ indicates we're using handle instead of publicId)
 * - /team/@handle for teams
 * - Various paths for resource types based on their specific type
 * 
 * @param objectType The type of the sitemap object (e.g., "ResourceVersion", "Team", "User").
 * @param properties The fetched properties of the object, used for URL disambiguation.
 * @returns The base URL path string for the given object type, or null if no match is found.
 */
function getLink(objectType: typeof sitemapObjectTypes[number], properties: SitemapProperties): string | null {
    switch (objectType) {
        case "ResourceVersion": {
            const resourceSubType = properties.resourceSubType;
            switch (resourceSubType) {
                case ResourceSubType.CodeDataConverter:
                    return LINKS.DataConverter;
                case ResourceSubType.CodeSmartContract:
                    return LINKS.SmartContract;
                case ResourceSubType.StandardPrompt:
                    return LINKS.Prompt;
                case ResourceSubType.StandardDataStructure:
                    return LINKS.DataStructure;
                case ResourceSubType.RoutineMultiStep:
                    return LINKS.RoutineMultiStep;
            }
            const resourceType = properties.root?.resourceType;
            if (resourceType === ResourceType.Api) {
                return LINKS.Api;
            } else if (resourceType === ResourceType.Note) {
                return LINKS.Note;
            } else if (resourceType === ResourceType.Project) {
                return LINKS.Project;
            } else if (resourceType === ResourceType.Routine) {
                return LINKS.RoutineSingleStep;
            }
            logger.warn(`No specific link found for ResourceVersion with subType: ${resourceSubType} and rootType: ${resourceType}, publicId: ${properties.publicId}`, { properties });
            return null;
        }
        case "Team":
            return LINKS.Team;
        case "User":
            return LINKS.User;
        default: {
            logger.error(`Unhandled objectType in getLink: ${objectType}`, { properties });
            return null;
        }
    }
}

// Where to save the sitemap index and files
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const sitemapIndexDir = path.resolve(__dirname, "../../../ui/dist");
const sitemapDir = `${sitemapIndexDir}/sitemaps`;

/**
 * Generates and saves one or more sitemap.xml.gz files for a specific database object type.
 * It fetches public objects of the given type in batches, converts them to sitemap entries,
 * and writes them to gzipped XML files. Each file adheres to sitemap limits 
 * (MAX_ENTRIES_PER_SITEMAP entries or MAX_SITEMAP_FILE_SIZE_BYTES).
 * 
 * This function is a core part of the dynamic sitemap generation process for user-generated content.
 *
 * @param objectType The database object type (e.g., "ResourceVersion", "Team", "User") to generate sitemaps for.
 * @returns A promise that resolves to an array of generated sitemap file names (e.g., ["User-0.xml.gz", "User-1.xml.gz"]). 
 *          Returns an empty array if no public objects of the type are found, to prevent empty sitemap files.
 */
async function genSitemapForObject(
    objectType: typeof sitemapObjectTypes[number],
): Promise<string[]> {
    logger.info(`Generating sitemap for ${objectType}`);
    // Initialize return value
    const sitemapFileNames: string[] = [];
    // For objects that support handles, we prioritize those urls over the id urls
    const supportsHandles = ["Team", "User"].includes(objectType);
    // Create do-while loop which breaks when the objects returned are less than the batch size
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Create another do-while loop which breaks when the collected entries are reaching the limit 
        // of a sitemap file (50,000 entries or 50MB)
        const collectedEntries: SitemapEntryContent[] = [];
        let estimatedFileSize = 0;
        do {
            // Find all public objects
            const { dbTable, validate } = ModelMap.getLogic(["dbTable", "validate"], objectType);
            const dbModel = DbProvider.get()[dbTable];
            
            if (!isPrismaDelegate(dbModel)) {
                logger.error(`Database model for ${objectType} is not a valid PrismaDelegate`, {
                    objectType,
                    dbTable,
                });
                throw new Error(`Invalid database model for ${objectType}`);
            }
            
            // Build where condition - manually add visibility filters since validate().visibility.public may not work in jobs context
            let visibilityFilter = {};
            
            // Apply type-specific visibility filters
            switch (objectType) {
                case "User":
                    visibilityFilter = { isPrivate: false, isBot: false };
                    break;
                case "Team":
                    visibilityFilter = { isPrivate: false };
                    break;
                case "ResourceVersion":
                    visibilityFilter = { 
                        isPrivate: false, 
                        isDeleted: false,
                        isComplete: true,
                        root: {
                            isPrivate: false,
                            isDeleted: false,
                        },
                    };
                    break;
                default: {
                    // Fallback to validate function if available
                    const validateVisibility = validate().visibility?.public;
                    if (validateVisibility) {
                        visibilityFilter = validateVisibility;
                    }
                }
            }
            
            const whereCondition = visibilityFilter;
            
            const batchResult = await dbModel.findMany({
                where: whereCondition,
                // Grab id, handle, root data, and translations
                select: {
                    id: true,
                    publicId: true,
                    ...(supportsHandles && { handle: true }),
                    translations: {
                        select: {
                            language: true,
                        },
                    },
                    // Object type-specific fields for disambiguating urls
                    ...(objectType === "ResourceVersion" && {
                        isDeleted: true,
                        resourceSubType: true,
                        versionLabel: true,
                        root: {
                            select: {
                                id: true,
                                publicId: true,
                                isDeleted: true,
                                isPrivate: true,
                                resourceType: true,
                            },
                        },
                    }),
                },
                skip,
                take: BATCH_SIZE_SMALL,
            });
            
            // Type validation and transformation with safer handling
            const batch: SitemapProperties[] = [];
            
            for (const item of batchResult) {
                try {
                    // Validate required fields exist
                    if (!item || typeof item !== "object") {
                        logger.warn(`Invalid item in batch result for ${objectType}`, { item });
                        continue;
                    }
                    
                    // Safe bigint to string conversion
                    const idStr = item.id ? item.id.toString() : "";
                    if (!idStr) {
                        logger.warn(`Item missing id in batch result for ${objectType}`, { item });
                        continue;
                    }
                    
                    // Create base object with required fields
                    const result: SitemapProperties = {
                        id: idStr,
                        publicId: typeof item.publicId === "string" ? item.publicId : "",
                        translations: Array.isArray(item.translations) ? item.translations : [],
                    };
                    
                    // Add optional fields with proper type validation
                    if (supportsHandles && hasProperty(item, "handle") && typeof item.handle === "string") {
                        result.handle = item.handle;
                    }
                    
                    if (hasProperty(item, "isDeleted") && typeof item.isDeleted === "boolean") {
                        result.isDeleted = item.isDeleted;
                    }
                    
                    if (hasProperty(item, "resourceSubType") && typeof item.resourceSubType === "string" && isValidResourceSubType(item.resourceSubType)) {
                        result.resourceSubType = item.resourceSubType;
                    }
                    
                    if (hasProperty(item, "versionLabel") && typeof item.versionLabel === "string") {
                        result.versionLabel = item.versionLabel;
                    }
                    
                    if (hasProperty(item, "root") && item.root && isValidRootObject(item.root)) {
                        result.root = item.root;
                    }
                    
                    batch.push(result);
                } catch (error: unknown) {
                    logger.error(`Error processing batch item for ${objectType}`, { error, item });
                }
            }
            // Increment skip
            skip += BATCH_SIZE_SMALL;
            // Update current batch size
            currentBatchSize = batch.length;
            // Safe calculation of base URL size - handle the case where objectType might not have a direct LINKS mapping
            const linkKey = objectType.replace("Version", "");
            let linkValue = "";
            if (isValidLinkKey(linkKey)) {
                linkValue = LINKS[linkKey];
            } else {
                // For types that don't have direct LINKS mapping, estimate with empty string
                logger.debug(`No LINKS mapping found for linkKey: ${linkKey} (from objectType: ${objectType})`);
            }
            const baseUrlSize = UI_URL_REMOTE.length + linkValue.length;
            for (const entry of batch) {
                // Convert batch to SiteMapEntryContent
                const objectLink = getLink(objectType, entry);

                if (objectLink === null) {
                    logger.info(`Skipping sitemap entry for ${objectType} publicId ${entry.publicId} due to missing link.`);
                    continue;
                }

                const entryContent: SitemapEntryContent = {
                    publicId: entry.publicId,
                    languages: entry.translations.map(translation => translation.language),
                    objectLink,
                };
                if (entry.handle) {
                    entryContent.handle = entry.handle;
                }
                if (entry.versionLabel) {
                    entryContent.versionLabel = entry.versionLabel;
                }
                if (entry.root?.handle) {
                    entryContent.rootHandle = entry.root.handle;
                }
                if (entry.root?.publicId) {
                    entryContent.rootPublicId = entry.root.publicId;
                }
                // Add entry to collected entries
                collectedEntries.push(entryContent);
                // Estimate bytes for entry, based on how many languages it has
                estimatedFileSize += (baseUrlSize + objectLink.length + entry.id.length) * entry.translations.length + ESTIMATED_ENTRY_OVERHEAD_BYTES;
            }
        } while (
            collectedEntries.length < MAX_ENTRIES_PER_SITEMAP &&
            estimatedFileSize < MAX_SITEMAP_FILE_SIZE_BYTES &&
            currentBatchSize === BATCH_SIZE_SMALL
        );
        // If no entries were collected, return an empty array. 
        // This is to prevent an empty sitemap file from being generated, which gives an error in Google Search Console
        if (collectedEntries.length === 0) {
            return [];
        }
        // Convert collected entries to sitemap file
        const sitemap = generateSitemap(UI_URL_REMOTE, { content: collectedEntries });
        // Zip and save sitemap file synchronously
        const sitemapFileName = `${objectType}-${sitemapFileNames.length}.xml.gz`;
        try {
            const buffer = zlib.gzipSync(sitemap);
            fs.writeFileSync(`${sitemapDir}/${sitemapFileName}`, buffer as unknown as Uint8Array);
            // Add sitemap file name to array only after successful write
            sitemapFileNames.push(sitemapFileName);
        } catch (error: unknown) {
            logger.error("Failed to create or write sitemap file", { sitemapFileName, error });
        }
    } while (currentBatchSize === BATCH_SIZE_SMALL);
    // Return sitemap file names
    return sitemapFileNames;
}

/**
 * Orchestrates the generation of the complete sitemap structure for the application.
 * This function performs the following steps:
 * 1. Ensures the target sitemap directories exist (`/sitemaps` within `ui/dist`).
 * 2. Checks for the presence of `sitemap-routes.xml` (expected to be generated by a UI build process for static routes).
 * 3. Initializes the `ModelMap` for database interactions.
 * 4. Iterates through `sitemapObjectTypes` (dynamic content like ResourceVersion, Team, User):
 *    - Calls `genSitemapForObject` for each type to generate individual gzipped sitemap files (e.g., `User-0.xml.gz`).
 * 5. Collects all generated sitemap file names.
 * 6. Generates a `sitemap.xml` index file that lists all individual sitemap files (both static and dynamic).
 * 7. Writes the main `sitemap.xml` to the `ui/dist` directory.
 *
 * This function is intended to be run as a scheduled job (e.g., cron) to keep the sitemap fresh,
 * reflecting the latest publicly available dynamic content.
 */
export async function genSitemap(): Promise<void> {
    // Ensure all directories exist
    if (!fs.existsSync(sitemapIndexDir)) {
        fs.mkdirSync(sitemapIndexDir, { recursive: true });
    }
    if (!fs.existsSync(sitemapDir)) {
        fs.mkdirSync(sitemapDir, { recursive: true });
    }
    // Check if sitemap file for main route exists
    const routeSitemapFileName = "sitemap-routes.xml";
    // Initialize ModelMap, which is needed later
    await ModelMap.init();
    try {
        const sitemapFileNames: string[] = [];
        // Generate sitemap for each object type
        for (const objectType of sitemapObjectTypes) {
            const sitemapFileNamesForObject = await genSitemapForObject(objectType);
            // Add sitemap file names to array
            sitemapFileNames.push(...sitemapFileNamesForObject);
        }

        const allSitemapFilesForIndex: string[] = [];
        if (fs.existsSync(`${sitemapDir}/${routeSitemapFileName}`)) {
            allSitemapFilesForIndex.push(routeSitemapFileName);
        } else {
            logger.warning("Sitemap file for main routes does not exist and will not be included in the index.", { trace: "0591", routeSitemapFileName, sitemapDir });
        }
        allSitemapFilesForIndex.push(...sitemapFileNames);

        // Generate sitemap index file
        const sitemapIndex = generateSitemapIndex(`${UI_URL_REMOTE}/sitemaps`, allSitemapFilesForIndex);
        // Write sitemap index file
        fs.writeFileSync(`${sitemapIndexDir}/sitemap.xml`, sitemapIndex);
        logger.info("✅ Sitemap generated successfully");
    } catch (error: unknown) {
        logger.error("genSitemap caught error", { error: error instanceof Error ? { message: error.message, name: error.name, stack: error.stack } : error, trace: "0463", routeSitemapFileName, sitemapDir });
    }
}

/**
 * Checks if the main sitemap index file (`sitemap.xml`) is missing from its expected location (`ui/dist`).
 * This is used by the cron job system to determine if an initial sitemap generation 
 * should be run immediately upon service startup if no sitemap is found.
 * 
 * @returns A promise that resolves to `true` if `sitemap.xml` does not exist, `false` otherwise.
 */
export async function isSitemapMissing(): Promise<boolean> {
    return !fs.existsSync(`${sitemapIndexDir}/sitemap.xml`);
}
