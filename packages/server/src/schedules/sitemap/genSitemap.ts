import { generateSitemap, SitemapEntry } from '@shared/utils';
import fs from 'fs';
import pkg from '@prisma/client';
import { GqlModelType } from '@shared/consts';
import { PrismaType } from '../../types';
import { getLogic } from '../../getters';
const { PrismaClient } = pkg;

type SitemapObjectTypes =  `${GqlModelType.Api | GqlModelType.Note | GqlModelType.Organization | GqlModelType.Project | GqlModelType.Question | GqlModelType.Routine | GqlModelType.SmartContract | GqlModelType.Standard | GqlModelType.User}`;

/**
 * Batch collects sitemap entries for an object
 * @param prisma The Prisma client
 * @param objectType The object type to collect sitemap entries for
 * @param objectIds The IDs of the objects to collect sitemap entries for
 * @returns An array of sitemap entries
 */
const batchCollectEntries = async (
    prisma: PrismaType,
    objectType: SitemapObjectTypes,
    objectIds: string[],
): Promise<string[]> => {
    // Initialize return value
    const result: string[] = [];
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all public objects
        const { delegate, validate } = getLogic(['delegate'], objectType, ['en'], 'batchCollectEntries')
        const batch = await delegate(prisma).findMany({
            where: {
                id: { in: objectIds },
                ...validate?.visibility.public(),
            },
            select: {
                id: true,
                //TODO
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // Convert batch to sitemap entries
        // const entries = batch.map(entry => {
        //     generateSitemap({

        //     })
        // Add batch to result
        // result.push(...batch);
    } while (currentBatchSize === batchSize);
    return result;
}