import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import { CODE, IMAGE_SIZE } from '@local/shared';
import { CustomError } from '../error';
import { deleteImage, saveImage } from '../utils';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Image;

export const typeDef = gql`
    enum ImageSize {
        XXS
        XS
        S
        M
        ML
        L
        XL
        XXL
    }

    input ImageUpdate {
        hash: String!
        alt: String
        description: String
    }

    type AddImageResponse {
        success: Boolean!
        src: String
        hash: String
    }

    type ImageFile {
        hash: String!
        src: String!
        width: Int!
        height: Int!
    }

    type Image {
        hash: String!
        alt: String
        description: String
        files: [ImageFile!]
    }

    extend type Query {
        imagesByLabel(label: String!): [Image!]!
    }

    extend type Mutation {
        addImages(
            files: [Upload!]!
            alts: [String]
            descriptions: [String]
            labels: [String!]
        ): [AddImageResponse!]!
        updateImages(
            data: [ImageUpdate!]!
            deleting: [String!]
            label: String
        ): Boolean!
        deleteImages(
            hashes: [String!]!
        ): Count!
        # Images with labels that are not in this request will be saved
        deleteImagesByLabel(
            labels: [String!]!
        ): Count!
    }
`

export const resolvers = {
    ImageSize: IMAGE_SIZE,
    Query: {
        imagesByLabel: async (_, args, context, info) => {
            // Get all images with label
            let images = await context.prisma[_model].findMany({
                where: { labels: { some: { label: args.label } } },
                select: { hash: true, labels: { select: { label: true, index: true } } }
            })
            // Sort by position
            images = images.sort((a, b) => {
                const aIndex = a.labels.find(l => l.label === args.label);
                const bIndex = b.labels.find(l => l.label === args.label);
                return aIndex > bIndex;
            })
            return await context.prisma[_model].findMany({ 
                where: { hash: { in: images.map(i => i.hash) } },
                ...(new PrismaSelect(info).value)
            });
        }
    },
    Mutation: {
        addImages: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            // Check for valid arguments
            // If alts provided, must match length of files
            if (args.alts && args.alts.length !== args.files.length) return new CustomError(CODE.InvalidArgs);
            let results = [];
            // Loop through every image passed in
            for (let i = 0; i < args.files.length; i++) {
                results.push(await saveImage({
                    file: args.files[i],
                    alt: args.alts ? args.alts[i] : undefined,
                    description: args.descriptions ? args.descriptions[i] : undefined,
                    labels: args.labels,
                    errorOnDuplicate: false
                }))
            }
            return results;
        },
        updateImages: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            // Loop through update data passed in
            for (let i = 0; i < args.data.length; i++) {
                const curr = args.data[i];
                if (args.label) {
                    // Update position in label
                    await context.prisma[TABLES.ImageLabels].update({
                        where: { image_labels_hash_label_unique: { hash: curr.hash, label: args.label } },
                        data: { index: i }
                    })
                }
                // Update alt and description
                await context.prisma[_model].update({
                    where: { hash: curr.hash },
                    data: { 
                        alt: curr.alt,
                        description: curr.description,
                    }
                })
            }
            if (!args.deleting) return true;
            // Loop through delete data passed in
            for (const hash of args.deleting) {
                await deleteImage(hash);
            }
            return true;
        },
        deleteImages: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            let count = 0;
            for (const hash of args.hashes) {
                if (await deleteImage(hash)) count++;
            }
            return count;
        },
        deleteImagesByLabel: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            const imagesToDelete = await context.prisma[TABLES.Image].findMany({
                where: { every: { label: { in: args.labels } } },
                select: {
                    hash: true
                }
            });
            await context.prisma[TABLES.ImageLabels].deleteMany({
                where: { label: { in: args.labels }}
            });
            let count = 0;
            for (const image of imagesToDelete) {
                if (await deleteImage(image.hash)) count++;
            }
            return count;
        },
    }
}