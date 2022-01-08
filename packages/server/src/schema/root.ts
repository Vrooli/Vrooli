import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo, GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
import ogs from 'open-graph-scraper';
import { AutocompleteInput, Autocomplete, OpenGraphResponse, OrganizationSortBy, ProjectSortBy, RoutineSortBy, UserSortBy, StandardSortBy } from './types';
import { AutocompleteResultType, MetricTimeFrame } from '@local/shared';
import { IWrap } from '../types';
import { Context } from '../context';
import { OrganizationModel, ProjectModel, RoutineModel, StandardModel, UserModel } from '../models';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    scalar Date
    scalar Upload

    # Enums for counting objects recently created or updated (for metrics)
    enum MetricTimeFrame {
        Daily
        Weekly
        Monthly
        Yearly
    }

    # Enums for autocomplete search result types
    enum AutocompleteResultType {
        Organization
        Project
        Routine
        Standard
        User
    }

    # Return type for a cursor-based pagination's pageInfo response
    type PageInfo {
        hasNextPage: Boolean!
        endCursor: String
    }
    # Return type for delete mutations,
    # which return the number of affected rows
    type Count {
        count: Int
    }
    # Return type for mutations with a success boolean
    # Could return just the boolean, but this makes it clear what the result means
    type Success {
        success: Boolean
    }
    # Return type for error messages
    type Response {
        code: Int
        message: String!
    }

    # Return type for Open Graph queries
    type OpenGraphResponse {
        site: String
        title: String
        description: String
        imageUrl: String
    }

    input ReadAssetsInput {
        files: [String!]!
    }

    input ReadOpenGraphInput {
        url: String!
    }

    input WriteAssetsInput {
        files: [Upload!]!
    }

    # Input for finding object by id
    input FindByIdInput {
        id: ID!
    }

    # Input for deleting one object
    input DeleteOneInput {
        id: ID!
    }

    # Input for deleting multiple objects
    input DeleteManyInput {
        ids: [ID!]!
    }

    # Input for reporting an object
    input ReportInput {
        id: ID!
        reason: String
    }

    # Input for site-wide autocomplete search
    input AutocompleteInput {
        searchString: String!
        take: Int
    }

    type Autocomplete {
        title: String!
        id: ID!
        objectType: AutocompleteResultType!
        stars: Int!
    }

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(input: ReadAssetsInput!): [String]!
        readOpenGraph(input: ReadOpenGraphInput!): OpenGraphResponse!
        autocomplete(input: AutocompleteInput!): [Autocomplete!]!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(input: WriteAssetsInput!): Boolean
    }
`

export const resolvers = {
    Upload: GraphQLUpload,
    MetricTimeFrame: MetricTimeFrame,
    AutocompleteResultType: AutocompleteResultType,
    Date: new GraphQLScalarType({
        name: "Date",
        description: "Custom description for the date scalar",
        // Assumes data is either Unix timestamp or Date object
        parseValue(value) {
            return new Date(value).toISOString(); // value from the client
        },
        serialize(value) {
            return new Date(value).getTime(); // value sent to the client
        },
        parseLiteral(ast: any) {
            return new Date(ast).toDateString(); // ast value is always in string format
        }
    }),
    Query: {
        readAssets: async (_parent: undefined, { input }: any): Promise<Array<String | null>> => {
            return await readFiles(input.files);
        },
        readOpenGraph: async (_parent: undefined, { input }: any): Promise<OpenGraphResponse> => {
            return await ogs({ url: input.url })
                .then((data: any) => {
                    const { result } = data;
                    return {
                        site: result?.ogSiteName,
                        title: result?.ogTitle,
                        description: result?.ogDescription,
                        imageUrl: result?.ogImage?.url,
                    };
                }).catch(err => {
                    console.error('Caught error fetching Open Graph url', err);
                    return {};
                }).finally(() => { return {} })
        },
        autocomplete: async (_parent: undefined, { input }: IWrap<AutocompleteInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<Autocomplete[]> => {
            const take = input.take ? Math.ceil(input.take / 4) : 10;
            //const MinimumStars = 1; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            //const starredByQuery = { starredBy: { gte: MinimumStars } }; TODO for now, Prisma does not offer this type of sorting. See https://github.com/prisma/prisma/issues/8935. Instead, returning if any stars exist.
            const starredByQuery = { NOT: { starredBy: { none: {} } } };
            // Query routines
            const routines = (await prisma.routine.findMany({
                where: {
                    ...RoutineModel().getSearchStringQuery(input.searchString),
                    ...starredByQuery,
                },
                orderBy: RoutineModel().getSortQuery(RoutineSortBy.StarsDesc),
                take,
                select: {
                    id: true,
                    title: true,
                    _count: { select: { starredBy: true } }
                }
            })).map((r: any) => ({
                id: r.id,
                title: r.title,
                objectType: AutocompleteResultType.Routine,
                stars: r._count.starredBy
            }));
            // Query organizations
            const organizations = (await prisma.organization.findMany({
                where: {
                    ...OrganizationModel().getSearchStringQuery(input.searchString),
                    ...starredByQuery,
                },
                orderBy: OrganizationModel().getSortQuery(OrganizationSortBy.StarsDesc),
                take,
                select: {
                    id: true,
                    name: true,
                    _count: { select: { starredBy: true } }
                }
            })).map((r: any) => ({
                id: r.id,
                title: r.name,
                objectType: AutocompleteResultType.Organization,
                stars: r._count.starredBy
            }));
            // Query projects
            const projects = (await prisma.project.findMany({
                where: {
                    ...ProjectModel().getSearchStringQuery(input.searchString),
                    ...starredByQuery,
                },
                orderBy: ProjectModel().getSortQuery(ProjectSortBy.StarsDesc),
                take,
                select: {
                    id: true,
                    name: true,
                    _count: { select: { starredBy: true } }
                }
            })).map((r: any) => ({
                id: r.id,
                title: r.name,
                objectType: AutocompleteResultType.Project,
                stars: r._count.starredBy
            }));
            // Query standards
            const standards = (await prisma.organization.findMany({
                where: {
                    ...StandardModel().getSearchStringQuery(input.searchString),
                    ...starredByQuery,
                },
                orderBy: StandardModel().getSortQuery(StandardSortBy.StarsDesc),
                take,
                select: {
                    id: true,
                    name: true,
                    _count: { select: { starredBy: true } }
                }
            })).map((r: any) => ({
                id: r.id,
                title: r.name,
                objectType: AutocompleteResultType.Standard,
                stars: r._count.starredBy
            }));
            // Query users
            const users = (await prisma.user.findMany({
                where: {
                    ...UserModel().getSearchStringQuery(input.searchString),
                    //...starredByQuery,
                },
                orderBy: UserModel().getSortQuery(UserSortBy.StarsDesc),
                take,
                select: {
                    id: true,
                    username: true,
                    _count: { select: { starredBy: true } }
                }
            })).map((r: any) => ({
                id: r.id,
                title: r.username,
                objectType: AutocompleteResultType.User,
                stars: r._count.starredBy
            }));
            console.log('USERS', users);
            // Combine query results and sort by stars
            return routines.concat(organizations, projects, standards, users).sort((a: any, b: any) => {
                return b.stars - a.stars;
            });
        },
    },
    Mutation: {
        writeAssets: async (_parent: undefined, { input }: any): Promise<boolean> => {
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}