import { generateSitemap, generateSitemapIndex, LINKS, SitemapEntryContent } from "@local/shared";
import fs from "fs";
import zlib from "zlib";
import { logger } from "../../events";
import { getLogic } from "../../getters";
import { PrismaType } from "../../types";
import { withPrisma } from "../../utils/withPrisma";

const sitemapObjectTypes = ["ApiVersion", "NoteVersion", "Organization", "ProjectVersion", "Question", "RoutineVersion", "SmartContractVersion", "StandardVersion", "User"] as const;

/**
 * Maps object types to their url base
 */
const Links: Record<typeof sitemapObjectTypes[number], string> = {
    ApiVersion: LINKS.Api,
    NoteVersion: LINKS.Note,
    Organization: LINKS.Organization,
    ProjectVersion: LINKS.Project,
    Question: LINKS.Question,
    RoutineVersion: LINKS.Routine,
    SmartContractVersion: LINKS.SmartContract,
    StandardVersion: LINKS.Standard,
    User: LINKS.User,
};

// Where to save the sitemap index and files
const sitemapIndexDir = "../../packages/ui/dist";
const sitemapDir = `${sitemapIndexDir}/sitemaps`;

// Name of website
const siteName = "https://vrooli.com";

/**
 * Converts a string to a BigInt
 * @param value String to convert
 * @param radix Radix (base) to use
 * @returns 
 */
function toBigInt(value: string, radix: number) {
    return [...value.toString()]
        .reduce((r, v) => r * BigInt(radix) + BigInt(parseInt(v, radix)), 0n);
}

/**
 * Converts a UUID into a shorter, base 36 string without dashes. 
 * Useful for displaying UUIDs in a more compact format, such as in a URL.
 * @param uuid v4 UUID to convert
 * @returns base 36 string without dashes
 */
const uuidToBase36 = (uuid: string): string => {
    try {
        const base36 = toBigInt(uuid.replace(/-/g, ""), 16).toString(36);
        return base36 === "0" ? "" : base36;
    } catch (error) {
        return uuid;
    }
};


/**
 * Generates and saves one or more sitemap files for an object
 * @param objectType The object type to collect sitemap entries for
 * @returns Names of the files that were generated
 */
const genSitemapForObject = async (
    prisma: PrismaType,
    objectType: typeof sitemapObjectTypes[number],
): Promise<string[]> => {
    logger.info(`Generating sitemap for ${objectType}`);
    // Initialize return value
    const sitemapFileNames: string[] = [];
    // For objects that support handles, we prioritize those urls over the id urls
    const supportsHandles = ["Organization", "ProjectVersion", "User"].includes(objectType);
    // For versioned objects, we also need to collect the root Id/handle
    const isVersioned = objectType.endsWith("Version");
    // If object can be private (in which case we don't want to include it in the sitemap)
    const canBePrivate = !["Question"].includes(objectType);
    // Create do-while loop which breaks when the objects returned are less than the batch size
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Create another do-while loop which breaks when the collected entries are reaching the limit 
        // of a sitemap file (50,000 entries or 50MB)
        const collectedEntries: SitemapEntryContent[] = [];
        let estimatedFileSize = 0;
        do {
            // Find all public objects
            const { delegate, validate } = getLogic(["delegate", "validate"], objectType, ["en"], "batchCollectEntries");
            const batch = await delegate(prisma).findMany({
                where: {
                    ...validate.visibility.public,
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
                },
                skip,
                take: batchSize,
            });
            // Increment skip
            skip += batchSize;
            // Update current batch size
            currentBatchSize = batch.length;
            const baseUrlSize = siteName.length + LINKS[objectType.replace("Version", "")].length;
            for (const entry of batch) {
                // Convert batch to SiteMapEntryContent
                const entryContent: SitemapEntryContent = {
                    handle: entry.handle,
                    id: uuidToBase36(entry.id),
                    languages: entry.translations.map(translation => translation.language),
                    objectLink: Links[objectType],
                    rootHandle: entry.root?.handle,
                    rootId: entry.root?.id ? uuidToBase36(entry.root.id) : undefined,
                };
                // Add entry to collected entries
                collectedEntries.push(entryContent);
                // Estimate bytes for entry, based on how many languages it has
                estimatedFileSize += (baseUrlSize + entry.id.length) * entry.translations.length + 100;
            }
        } while (collectedEntries.length < 50000 && estimatedFileSize < 50000000 && currentBatchSize === batchSize);
        // If no entries were collected, return an empty array. 
        // This is to prevent an empty sitemap file from being generated, which gives an error in Google Search Console
        if (collectedEntries.length === 0) {
            return [];
        }
        // Convert collected entries to sitemap file
        const sitemap = generateSitemap(siteName, { content: collectedEntries });
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
    } while (currentBatchSize === batchSize);
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
        logger.warning("Sitemap file for main routes does not exist");
    }
    const success = await withPrisma({
        process: async (prisma) => {
            const sitemapFileNames: string[] = [];
            // Generate sitemap for each object type
            for (const objectType of sitemapObjectTypes) {
                const sitemapFileNamesForObject = await genSitemapForObject(prisma, objectType);
                // Add sitemap file names to array
                sitemapFileNames.push(...sitemapFileNamesForObject);
            }
            // Generate sitemap index file
            const sitemapIndex = generateSitemapIndex(`${siteName}/sitemaps`, [routeSitemapFileName, ...sitemapFileNames]);
            // Write sitemap index file
            fs.writeFileSync(`${sitemapIndexDir}/sitemap.xml`, sitemapIndex);
        },
        trace: "0463",
        traceObject: { routeSitemapFileName, sitemapDir },
    });
    if (success) logger.info("✅ Sitemap generated successfully");
};

/**
 * Calls genSitemap if no sitemap.xml file exists
 */
export const genSitemapIfNotExists = async (): Promise<void> => {
    if (!fs.existsSync(`${sitemapIndexDir}/sitemap.xml`)) {
        await genSitemap();
    }
};
