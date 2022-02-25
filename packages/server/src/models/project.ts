import { CODE, MemberRole, projectCreate, projectUpdate } from "@local/shared";
import { CustomError } from "../error";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { DeleteOneInput, FindByIdInput, Project, ProjectCountInput, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy, Success } from "../schema/types";
import { addCreatorField, addJoinTables, addOwnerField, counter, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipFormatter, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { ResourceModel } from "./resource";
import { project } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import _ from "lodash";

export const projectDBFields = ['id', 'created_at', 'updated_at', 'description', 'name', 'score', 'stars', 'createdByUserId', 'createdByOrganizationId', 'userId', 'organizationId', 'parentId'];

//==============================================================
/* #region Custom Components */
//==============================================================

type ProjectFormatterType = FormatConverter<Project, project>;
/**
 * Component for formatting between graphql and prisma types
 */
export const projectFormatter = (): ProjectFormatterType => {
    const joinMapper = {
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
        starredBy: 'user',
    };
    return {
        dbShape: (partial: PartialSelectConvert<Project>): PartialSelectConvert<project> => {
            let modified = partial;
            console.log('project toDB', modified);
            // Deconstruct GraphQL unions
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbShape],
                ['forks', FormatterMap.Project.dbShape],
                ['parent', FormatterMap.Project.dbShape],
                ['reports', FormatterMap.Report.dbShape],
                ['resources', FormatterMap.Resource.dbShape],
                ['routines', FormatterMap.Routine.dbShape],
                ['starredBy', FormatterMap.User.dbShapeUser],
                ['tags', FormatterMap.Tag.dbShape],
            ]);
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<project> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Remove calculated fields
            let { isUpvoted, isStarred, role, ...rest } = modified;
            modified = rest;
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbPrune],
                ['forks', FormatterMap.Project.dbPrune],
                ['parent', FormatterMap.Project.dbPrune],
                ['reports', FormatterMap.Report.dbPrune],
                ['resources', FormatterMap.Resource.dbPrune],
                ['routines', FormatterMap.Routine.dbPrune],
                ['starredBy', FormatterMap.User.dbPruneUser],
                ['tags', FormatterMap.Tag.dbPrune],
            ]);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<project> => {
            return projectFormatter().dbShape(projectFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<project>): RecursivePartial<Project> => {
            if (!_.isObject(obj)) return obj;
            console.log('projectFormatter.toGraphQL start', obj);
            // Create unions
            let modified = addCreatorField(obj);
            modified = addOwnerField(modified);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            modified = removeJoinTables(modified, joinMapper);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.selectToGraphQL],
                ['forks', FormatterMap.Project.selectToGraphQL],
                ['parent', FormatterMap.Project.selectToGraphQL],
                ['reports', FormatterMap.Report.selectToGraphQL],
                ['resources', FormatterMap.Resource.selectToGraphQL],
                ['routines', FormatterMap.Routine.selectToGraphQL],
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
                ['tags', FormatterMap.Tag.selectToGraphQL],
            ]);
            console.log('projectFormatter.toGraphQL finished', modified);
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
export const projectSorter = (): Sortable<ProjectSortBy> => ({
    defaultSort: ProjectSortBy.DateUpdatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ProjectSortBy.AlphabeticalAsc]: { name: 'asc' },
            [ProjectSortBy.AlphabeticalDesc]: { name: 'desc' },
            [ProjectSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [ProjectSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [ProjectSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [ProjectSortBy.ForksDesc]: { forks: { _count: 'desc' } },
            [ProjectSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ProjectSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ProjectSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ProjectSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ProjectSortBy.StarsAsc]: { stars: 'asc' },
            [ProjectSortBy.StarsDesc]: { stars: 'desc' },
            [ProjectSortBy.VotesAsc]: { score: 'asc' },
            [ProjectSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { name: { ...insensitive } },
                { description: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    }
})

/**
 * Handles the authorized adding, updating, and deleting of projects.
 */
const projecter = (format: ProjectFormatterType, sort: Sortable<ProjectSortBy>, prisma: PrismaType) => ({
    async find(
        userId: string | null | undefined, // Of the user making the request, not the project
        input: FindByIdInput,
        info: InfoType,
    ): Promise<RecursivePartial<Project> | null> {
        // Create selector
        const select = selectHelper(info, format.selectToDB);
        // Access database
        let project = await prisma.project.findUnique({ where: { id: input.id }, ...select });
        // Return project with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!project) throw new CustomError(CODE.InternalError, 'Project not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [project], info);
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: ProjectSearchInput,
        info: InfoType,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const parentIdQuery = input.parentId ? { parentId: input.parentId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        // Search
        const search = searcher<ProjectSortBy, ProjectSearchInput, Project, project>(MODEL_TYPES.Project, format.selectToDB, sort, prisma);
        console.log('calling project search...')
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, info);
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: ProjectCreateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Project>> {
        // Check for valid arguments
        projectCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create project data
        let projectData: { [x: string]: any } = {
            name: input.name,
            description: input.description,
            parentId: input.parentId,
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, true),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, true),
        };
        // Associate with either organization or user
        if (input.createdByOrganizationId) {
            // Make sure the user is an admin of the organization
            const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            projectData = {
                ...projectData,
                organization: { connect: { id: input.createdByOrganizationId } },
                createdByOrganization: { connect: { id: input.createdByOrganizationId } },
            };
        } else {
            projectData = {
                ...projectData,
                user: { connect: { id: userId } },
                createdByUser: { connect: { id: userId } },
            };
        }
        console.log('creating project', projectData);
        // Create project
        const project = await prisma.project.create({
            data: projectData as any,
            ...selectHelper(info, format.selectToDB)
        })
        // Return project with "role", "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.selectToGraphQL(project), role: MemberRole.Owner as any, isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: ProjectUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Project>> {
        // Check for valid arguments
        projectUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create project data
        let projectData: { [x: string]: any } = {
            name: input.name,
            description: input.description,
            parentId: input.parentId,
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, false),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, false),
        };
        // Associate with either organization or user. This will remove the association with the other.
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            projectData = {
                ...projectData,
                organization: { connect: { id: input.organizationId } },
                user: { disconnect: true },
            };
        } else {
            projectData = {
                ...projectData,
                user: { connect: { id: userId } },
                organization: { disconnect: true },
            };
        }
        // Find project
        let project = await prisma.project.findFirst({
            where: {
                AND: [
                    { id: input.id },
                    {
                        OR: [
                            { organizationId: input.organizationId },
                            { userId },
                        ]
                    }
                ]
            }
        })
        if (!project) throw new CustomError(CODE.ErrorUnknown);
        // Update project
        project = await prisma.project.update({
            where: { id: project.id },
            data: projectData as any,
            ...selectHelper(info, format.selectToDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [project], info);
        return formatted[0];
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const project = await prisma.project.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                userId: true,
                organizationId: true,
            }
        })
        if (!project) throw new CustomError(CODE.NotFound, "Project not found");
        // Check if user is authorized
        let authorized = userId === project.userId;
        if (!authorized && project.organizationId) {
            [authorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, project.organizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.project.delete({
            where: { id: project.id },
        });
        return { success: true };
    },
    /**
     * Supplemental fields are role, isUpvoted, and isStarred
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Project>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'project')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, 'project')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for role
        if (partial.role) {
            console.log('project supplemental fields', objects[0].owner)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => x.owner?.__typename === 'Organization')
                .map(x => x.id)
                .filter(x => Boolean(x)) as string[];
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: Boolean(userId) && x.owner?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
    },
    profanityCheck(data: ProjectCreateInput | ProjectUpdateInput): void {
        if (hasProfanity(data.name, data.description)) throw new CustomError(CODE.BannedWord);
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ProjectModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Project;
    const format = projectFormatter();
    const sort = projectSorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<ProjectCountInput>(model, prisma),
        ...projecter(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================