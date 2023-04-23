import { LINKS } from "@local/consts";
import { generateSitemap, generateSitemapIndex } from "@local/utils";
import pkg from "@prisma/client";
import fs from "fs";
import zlib from "zlib";
import { logger } from "../../events";
import { getLogic } from "../../getters";
const { PrismaClient } = pkg;
const sitemapObjectTypes = ["ApiVersion", "NoteVersion", "Organization", "ProjectVersion", "Question", "RoutineVersion", "SmartContractVersion", "StandardVersion", "User"];
const Links = {
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
const sitemapIndexDir = "../../packages/ui/public";
const sitemapDir = `${sitemapIndexDir}/sitemaps`;
const siteName = "https://vrooli.com";
const genSitemapForObject = async (prisma, objectType) => {
    logger.info(`Generating sitemap for ${objectType}`);
    const sitemapFileNames = [];
    const supportsHandles = ["Organization", "ProjectVersion", "User"].includes(objectType);
    const isVersioned = objectType.endsWith("Version");
    const canBePrivate = !["Question"].includes(objectType);
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const collectedEntries = [];
        let estimatedFileSize = 0;
        do {
            const { delegate, validate } = getLogic(["delegate", "validate"], objectType, ["en"], "batchCollectEntries");
            const batch = await delegate(prisma).findMany({
                where: {
                    ...validate.visibility.public,
                },
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
            skip += batchSize;
            currentBatchSize = batch.length;
            const baseUrlSize = siteName.length + LINKS[objectType.replace("Version", "")].length;
            for (const entry of batch) {
                const entryContent = {
                    handle: entry.handle,
                    id: entry.id,
                    languages: entry.translations.map(translation => translation.language),
                    objectLink: Links[objectType],
                    rootHandle: entry.root?.handle,
                    rootId: entry.root?.id,
                };
                collectedEntries.push(entryContent);
                estimatedFileSize += (baseUrlSize + entry.id.length) * entry.translations.length + 100;
            }
        } while (collectedEntries.length < 50000 && estimatedFileSize < 50000000 && currentBatchSize === batchSize);
        const sitemap = generateSitemap(siteName, { content: collectedEntries });
        const sitemapFileName = `${objectType}-${sitemapFileNames.length}.xml.gz`;
        zlib.gzip(sitemap, (err, buffer) => {
            if (err) {
                logger.error(err);
            }
            else {
                fs.writeFileSync(`${sitemapDir}/${sitemapFileName}`, buffer);
            }
        });
        sitemapFileNames.push(sitemapFileName);
    } while (currentBatchSize === batchSize);
    return sitemapFileNames;
};
export const genSitemap = async () => {
    if (!fs.existsSync(sitemapDir)) {
        fs.mkdirSync(sitemapDir);
    }
    const routeSitemapFileName = "sitemap-routes.xml";
    if (!fs.existsSync(`${sitemapDir}/${routeSitemapFileName}`)) {
        logger.warning("Sitemap file for main routes does not exist");
    }
    const prisma = new PrismaClient();
    const sitemapFileNames = [];
    for (const objectType of sitemapObjectTypes) {
        const sitemapFileNamesForObject = await genSitemapForObject(prisma, objectType);
        sitemapFileNames.push(...sitemapFileNamesForObject);
    }
    const sitemapIndex = generateSitemapIndex(`${siteName}/sitemaps`, [routeSitemapFileName, ...sitemapFileNames]);
    fs.writeFileSync(`${sitemapIndexDir}/sitemap.xml`, sitemapIndex);
    await prisma.$disconnect();
    console.info("✅ Sitemap generated successfully");
};
export const genSitemapIfNotExists = async () => {
    if (!fs.existsSync(`${sitemapIndexDir}/sitemap.xml`)) {
        await genSitemap();
    }
};
//# sourceMappingURL=genSitemap.js.map