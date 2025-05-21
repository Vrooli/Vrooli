import { DbProvider, ModelMap, type PrismaDelegate, UI_URL_REMOTE, logger } from "@local/server";
import { LINKS, ResourceSubType, ResourceType, type SitemapEntryContent, generateSitemap, generateSitemapIndex } from "@local/shared";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import zlib from "zlib";

const sitemapObjectTypes = [
    "ResourceVersion",
    "Team",
    "User",
] as const;

const MAX_ENTRIES_PER_SITEMAP = 50000;
const BYTES_PER_KILOBYTE = 1024;
const SITEMAP_SIZE_LIMIT_MB = 50;
const MAX_SITEMAP_FILE_SIZE_BYTES = SITEMAP_SIZE_LIMIT_MB * BYTES_PER_KILOBYTE * BYTES_PER_KILOBYTE; // 50MB in bytes
const ESTIMATED_ENTRY_OVERHEAD_BYTES = 100; // Estimated XML overhead per entry
const BATCH_SIZE = 100; // How many objects to fetch at a time

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
 * @param objectType The type of the sitemap object (e.g., "ResourceVersion", "Team", "User").
 * @param properties The fetched properties of the object, used for URL disambiguation.
 * @returns The base URL path string for the given object type, or an empty string if no match is found.
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
            const batch = await (DbProvider.get()[dbTable] as PrismaDelegate).findMany({
                where: {
                    ...validate().visibility.public,
                },
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
                                handle: true,
                            },
                        },
                    }),
                },
                skip,
                take: BATCH_SIZE,
            }) as SitemapProperties[];
            // Increment skip
            skip += BATCH_SIZE;
            // Update current batch size
            currentBatchSize = batch.length;
            const baseUrlSize = UI_URL_REMOTE.length + LINKS[objectType.replace("Version", "")].length;
            for (const entry of batch) {
                // Convert batch to SiteMapEntryContent
                const objectLink = getLink(objectType, entry);

                if (objectLink === null) {
                    logger.info(`Skipping sitemap entry for ${objectType} publicId ${entry.publicId} due to missing link.`);
                    continue;
                }

                const entryContent: SitemapEntryContent = {
                    handle: entry.handle,
                    publicId: entry.publicId,
                    versionLabel: entry.versionLabel,
                    languages: entry.translations.map(translation => translation.language),
                    objectLink,
                    rootHandle: entry.root?.handle,
                    rootPublicId: entry.root?.publicId,
                };
                // Add entry to collected entries
                collectedEntries.push(entryContent);
                // Estimate bytes for entry, based on how many languages it has
                estimatedFileSize += (baseUrlSize + objectLink.length + entry.id.length) * entry.translations.length + ESTIMATED_ENTRY_OVERHEAD_BYTES;
            }
        } while (
            collectedEntries.length < MAX_ENTRIES_PER_SITEMAP &&
            estimatedFileSize < MAX_SITEMAP_FILE_SIZE_BYTES &&
            currentBatchSize === BATCH_SIZE
        );
        // If no entries were collected, return an empty array. 
        // This is to prevent an empty sitemap file from being generated, which gives an error in Google Search Console
        if (collectedEntries.length === 0) {
            return [];
        }
        // Convert collected entries to sitemap file
        const sitemap = generateSitemap(UI_URL_REMOTE, { content: collectedEntries });
        // Zip and save sitemap file
        const sitemapFileName = `${objectType}-${sitemapFileNames.length}.xml.gz`;
        zlib.gzip(sitemap, (err, buffer) => {
            if (err) {
                logger.error(err);
            } else {
                fs.writeFileSync(`${sitemapDir}/${sitemapFileName}`, new Uint8Array(buffer));
            }
        });
        //fs.writeFileSync(`${sitemapDir}/${sitemapFileName}`, sitemap);
        // Add sitemap file name to array
        sitemapFileNames.push(sitemapFileName);
    } while (currentBatchSize === BATCH_SIZE);
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
    // Check if sitemap directory exists
    if (!fs.existsSync(sitemapDir)) {
        fs.mkdirSync(sitemapDir);
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
    } catch (error) {
        logger.error("genSitemap caught error", { error, trace: "0463", routeSitemapFileName, sitemapDir });
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
