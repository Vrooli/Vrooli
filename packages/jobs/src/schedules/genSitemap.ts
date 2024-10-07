import { logger, ModelMap, PrismaDelegate, prismaInstance, UI_URL_REMOTE } from "@local/server";
import { CodeType, generateSitemap, generateSitemapIndex, LINKS, SitemapEntryContent, StandardType, uuidToBase36 } from "@local/shared";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import zlib from "zlib";

const sitemapObjectTypes = [
    "ApiVersion",
    "CodeVersion",
    "NoteVersion",
    "ProjectVersion",
    "Question",
    "RoutineVersion",
    "StandardVersion",
    "Team",
    "User",
] as const;

const MAX_ENTRIES_PER_SITEMAP = 50000;
const MAX_SITEMAP_FILE_SIZE_BYTES = 50 * 1024 * 1024; // 50MB in bytes
const ESTIMATED_ENTRY_OVERHEAD_BYTES = 100; // Estimated XML overhead per entry
const BATCH_SIZE = 100; // How many objects to fetch at a time

/**
 * Maps object types to their url base
 */
const getLink = (objectType: typeof sitemapObjectTypes[number], properties: any): string => {
    switch (objectType) {
        case "CodeVersion":
            return properties.codeType === CodeType.DataConvert ? LINKS.DataConverter : LINKS.SmartContract;
        case "StandardVersion":
            return properties.variant === StandardType.Prompt ? LINKS.Prompt : LINKS.DataStructure;
        case "ApiVersion":
            return LINKS.Api;
        case "NoteVersion":
            return LINKS.Note;
        case "ProjectVersion":
            return LINKS.Project;
        case "Question":
            return LINKS.Question;
        case "RoutineVersion":
            return LINKS.Routine;
        case "Team":
            return LINKS.Team;
        case "User":
            return LINKS.User;
        default:
            return "";
    }
};

// Where to save the sitemap index and files
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const sitemapIndexDir = path.resolve(__dirname, "../../../ui/dist");
const sitemapDir = `${sitemapIndexDir}/sitemaps`;

/**
 * Generates and saves one or more sitemap files for an object
 * @param objectType The object type to collect sitemap entries for
 * @returns Names of the files that were generated
 */
const genSitemapForObject = async (
    objectType: typeof sitemapObjectTypes[number],
): Promise<string[]> => {
    logger.info(`Generating sitemap for ${objectType}`);
    // Initialize return value
    const sitemapFileNames: string[] = [];
    // For objects that support handles, we prioritize those urls over the id urls
    const supportsHandles = ["Team", "ProjectVersion", "User"].includes(objectType);
    // For versioned objects, we also need to collect the root Id/handle
    const isVersioned = objectType.endsWith("Version");
    // If object can be private (in which case we don't want to include it in the sitemap)
    const canBePrivate = !["Question"].includes(objectType);
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
            const batch = await (prismaInstance[dbTable] as PrismaDelegate).findMany({
                where: {
                    ...validate().visibility.public,
                },
                // Grab id, handle, root data, and translations
                select: {
                    id: true,
                    ...(canBePrivate && { isPrivate: true }),
                    ...(!isVersioned && supportsHandles && { handle: true }),
                    ...(isVersioned && {
                        root: {
                            select: {
                                id: true,
                                ...(canBePrivate && { isPrivate: true }),
                                ...(supportsHandles && { handle: true }),
                            },
                        },
                    }),
                    translations: {
                        select: {
                            language: true,
                        },
                    },
                    // Object type-specific fields for disambiguating urls
                    ...(objectType === "CodeVersion" && { codeType: true }),
                    ...(objectType === "StandardVersion" && { variant: true }),
                },
                skip,
                take: BATCH_SIZE,
            });
            // Increment skip
            skip += BATCH_SIZE;
            // Update current batch size
            currentBatchSize = batch.length;
            const baseUrlSize = UI_URL_REMOTE.length + LINKS[objectType.replace("Version", "")].length;
            for (const entry of batch) {
                // Convert batch to SiteMapEntryContent
                const objectLink = getLink(objectType, entry);
                const entryContent: SitemapEntryContent = {
                    handle: entry.handle,
                    id: uuidToBase36(entry.id),
                    languages: entry.translations.map(translation => translation.language),
                    objectLink,
                    rootHandle: entry.root?.handle,
                    rootId: entry.root?.id ? uuidToBase36(entry.root.id) : undefined,
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
                fs.writeFileSync(`${sitemapDir}/${sitemapFileName}`, buffer);
            }
        });
        //fs.writeFileSync(`${sitemapDir}/${sitemapFileName}`, sitemap);
        // Add sitemap file name to array
        sitemapFileNames.push(sitemapFileName);
    } while (currentBatchSize === BATCH_SIZE);
    // Return sitemap file names
    return sitemapFileNames;
};

/**
 * Generates sitemap index file and calls genSitemapForObject for each object type
 */
export const genSitemap = async (): Promise<void> => {
    // Check if sitemap directory exists
    if (!fs.existsSync(sitemapDir)) {
        fs.mkdirSync(sitemapDir);
    }
    // Check if sitemap file for main route exists
    const routeSitemapFileName = "sitemap-routes.xml";
    if (!fs.existsSync(`${sitemapDir}/${routeSitemapFileName}`)) {
        logger.warning("Sitemap file for main routes does not exist", { trace: "0591", routeSitemapFileName, sitemapDir });
    }
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
        // Generate sitemap index file
        const sitemapIndex = generateSitemapIndex(`${UI_URL_REMOTE}/sitemaps`, [routeSitemapFileName, ...sitemapFileNames]);
        // Write sitemap index file
        fs.writeFileSync(`${sitemapIndexDir}/sitemap.xml`, sitemapIndex);
        logger.info("✅ Sitemap generated successfully");
    } catch (error) {
        logger.error("genSitemap caught error", { error, trace: "0463", routeSitemapFileName, sitemapDir });
    }
};

export const isSitemapMissing = async (): Promise<boolean> => {
    return !fs.existsSync(`${sitemapIndexDir}/sitemap.xml`);
};
