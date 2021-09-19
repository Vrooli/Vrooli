import { gql } from 'apollo-server-express';
import { db, TABLES } from '../db';

const _model = TABLES.ProductTrait;

export const typeDef = gql`
    type ProductTrait {
        id: ID!
        name: String!
        value: String!
    }

    type TraitOptions {
        name: String!
        values: [String!]!
    }

    extend type Query {
        traitNames: [String!]!
        traitValues(name: String!): [String!]!
        traitOptions: [TraitOptions!]!
    }
`

export const resolvers = {
    Query: {
        traitNames: async () => {
            return await db(TABLES.ProductTrait).select('name').unique();
        },
        traitValues: async (_, args) => {
            return await db(TABLES.ProductTrait).select('value').where('name', args.name)
        },
        // Returns all values previously entered for every trait
        traitOptions: async (_, _args, context) => {
            // Query all data
            const trait_data = await context.prisma[_model].findMany();
            // Combine data into object
            let options = {};
            for (const row of trait_data) {
                options[row.name] ? options[row.name].push(row.value) : options[row.name] = [row.value];
            }
            // Format object
            let trait_options = []
            for (const [key, value] of Object.entries(options)) {
                trait_options.push({ name: key, values: [...new Set(value)] });
            }
            return trait_options;
        }
    },
}