import { ViewFor, ViewSortBy } from "@local/consts";
import { ApiModel, IssueModel, NoteModel, PostModel, QuestionModel, SmartContractModel } from ".";
import { onlyValidIds, selPad } from "../builders";
import { CustomError } from "../events";
import { getLabels, getLogic } from "../getters";
import { initializeRedis } from "../redisConn";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { UserModel } from "./user";
const toWhere = (key, nestedKey, id) => {
    if (nestedKey)
        return { [key]: { [nestedKey]: { some: { id } } } };
    return { [key]: { id } };
};
const toSelect = (key) => {
    if (key)
        return { [key]: { select: { id: true, views: true } } };
    return { id: true, views: true };
};
const toData = (object, key) => {
    if (key)
        return object[key];
    return object;
};
const toCreate = (object, relName, key) => {
    if (key)
        return { [relName]: { connect: { id: object[key].id } } };
    return { [relName]: { connect: { id: object.id } } };
};
const whereMapper = {
    Api: (id) => toWhere("api", null, id),
    ApiVersion: (id) => toWhere("api", "versions", id),
    Note: (id) => toWhere("note", null, id),
    NoteVersion: (id) => toWhere("note", "versions", id),
    Organization: (id) => toWhere("organization", null, id),
    Project: (id) => toWhere("project", null, id),
    ProjectVersion: (id) => toWhere("project", "versions", id),
    Question: (id) => toWhere("question", null, id),
    Routine: (id) => toWhere("routine", null, id),
    RoutineVersion: (id) => toWhere("routine", "versions", id),
    SmartContract: (id) => toWhere("smartContract", null, id),
    SmartContractVersion: (id) => toWhere("smartContract", "versions", id),
    Standard: (id) => toWhere("standard", null, id),
    StandardVersion: (id) => toWhere("standard", "versions", id),
    User: (id) => toWhere("user", null, id),
};
const selectMapper = {
    Api: toSelect(),
    ApiVersion: toSelect("root"),
    Note: toSelect(),
    NoteVersion: toSelect("root"),
    Organization: toSelect(),
    Project: toSelect(),
    ProjectVersion: toSelect("root"),
    Question: toSelect(),
    Routine: toSelect(),
    RoutineVersion: toSelect("root"),
    SmartContract: toSelect(),
    SmartContractVersion: toSelect("root"),
    Standard: toSelect(),
    StandardVersion: toSelect("root"),
    User: toSelect(),
};
const dataMapper = {
    Api: (object) => toData(object),
    ApiVersion: (object) => toData(object, "root"),
    Note: (object) => toData(object),
    NoteVersion: (object) => toData(object, "root"),
    Organization: (object) => toData(object),
    Project: (object) => toData(object),
    ProjectVersion: (object) => toData(object, "root"),
    Question: (object) => toData(object),
    Routine: (object) => toData(object),
    RoutineVersion: (object) => toData(object, "root"),
    SmartContract: (object) => toData(object),
    SmartContractVersion: (object) => toData(object, "root"),
    Standard: (object) => toData(object),
    StandardVersion: (object) => toData(object, "root"),
    User: (object) => toData(object),
};
const createMapper = {
    Api: (object) => toCreate(object, "api"),
    ApiVersion: (object) => toCreate(object, "api", "root"),
    Note: (object) => toCreate(object, "note"),
    NoteVersion: (object) => toCreate(object, "note", "root"),
    Organization: (object) => toCreate(object, "organization"),
    Project: (object) => toCreate(object, "project"),
    ProjectVersion: (object) => toCreate(object, "project", "root"),
    Question: (object) => toCreate(object, "question"),
    Routine: (object) => toCreate(object, "routine"),
    RoutineVersion: (object) => toCreate(object, "routine", "root"),
    SmartContract: (object) => toCreate(object, "smartContract"),
    SmartContractVersion: (object) => toCreate(object, "smartContract", "root"),
    Standard: (object) => toCreate(object, "standard"),
    StandardVersion: (object) => toCreate(object, "standard", "root"),
    User: (object) => toCreate(object, "user"),
};
const deleteViews = async (prisma, userId, ids) => {
    return await prisma.view.deleteMany({
        where: {
            AND: [
                { id: { in: ids } },
                { byId: userId },
            ],
        },
    }).then(({ count }) => ({ __typename: "Count", count }));
};
const clearViews = async (prisma, userId) => {
    return await prisma.view.deleteMany({
        where: { byId: userId },
    }).then(({ count }) => ({ __typename: "Count", count }));
};
const __typename = "View";
const suppFields = [];
export const ViewModel = ({
    __typename,
    delegate: (prisma) => prisma.view,
    display: {
        select: () => ({
            id: true,
            api: selPad(ApiModel.display.select),
            organization: selPad(OrganizationModel.display.select),
            question: selPad(QuestionModel.display.select),
            note: selPad(NoteModel.display.select),
            post: selPad(PostModel.display.select),
            project: selPad(ProjectModel.display.select),
            routine: selPad(RoutineModel.display.select),
            smartContract: selPad(SmartContractModel.display.select),
            standard: selPad(StandardModel.display.select),
            user: selPad(UserModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api)
                return ApiModel.display.label(select.api, languages);
            if (select.organization)
                return OrganizationModel.display.label(select.organization, languages);
            if (select.question)
                return QuestionModel.display.label(select.question, languages);
            if (select.note)
                return NoteModel.display.label(select.note, languages);
            if (select.post)
                return PostModel.display.label(select.post, languages);
            if (select.project)
                return ProjectModel.display.label(select.project, languages);
            if (select.routine)
                return RoutineModel.display.label(select.routine, languages);
            if (select.smartContract)
                return SmartContractModel.display.label(select.smartContract, languages);
            if (select.standard)
                return StandardModel.display.label(select.standard, languages);
            if (select.user)
                return UserModel.display.label(select.user, languages);
            return "";
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            by: "User",
            to: {
                api: "Api",
                issue: "Issue",
                note: "Note",
                organization: "Organization",
                post: "Post",
                project: "Project",
                question: "Question",
                routine: "Routine",
                smartContract: "SmartContract",
                standard: "Standard",
                user: "User",
            },
        },
        prismaRelMap: {
            __typename,
            by: "User",
            api: "Api",
            issue: "Issue",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            user: "User",
        },
        countFields: {},
    },
    search: {
        defaultSort: ViewSortBy.LastViewedDesc,
        sortBy: ViewSortBy,
        searchFields: {
            lastViewedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                { api: ApiModel.search.searchStringQuery() },
                { issue: IssueModel.search.searchStringQuery() },
                { note: NoteModel.search.searchStringQuery() },
                { organization: OrganizationModel.search.searchStringQuery() },
                { question: QuestionModel.search.searchStringQuery() },
                { post: PostModel.search.searchStringQuery() },
                { project: ProjectModel.search.searchStringQuery() },
                { routine: RoutineModel.search.searchStringQuery() },
                { smartContract: SmartContractModel.search.searchStringQuery() },
                { standard: StandardModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
    },
    query: {
        async getIsVieweds(prisma, userId, ids, viewFor) {
            const result = new Array(ids.length).fill(false);
            if (!userId)
                return result;
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${viewFor.toLowerCase()}Id`;
            const isViewedArray = await prisma.view.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
            for (let i = 0; i < ids.length; i++) {
                result[i] = Boolean(isViewedArray.find((view) => view[fieldName] === ids[i]));
            }
            return result;
        },
    },
    view: async (prisma, userData, input) => {
        const { delegate } = getLogic(["delegate"], input.viewFor, userData.languages, "ViewModel.view");
        const objectToView = await delegate(prisma).findUnique({
            where: { id: input.forId },
            select: selectMapper[input.viewFor],
        });
        if (!objectToView)
            throw new CustomError("0173", "NotFound", userData.languages);
        let view = await prisma.view.findFirst({
            where: {
                by: { id: userData.id },
                ...whereMapper[input.viewFor](input.forId),
            },
        });
        if (view) {
            await prisma.view.update({
                where: { id: view.id },
                data: {
                    lastViewedAt: new Date(),
                },
            });
        }
        else {
            const labels = await getLabels([{ id: input.forId, languages: userData.languages }], input.viewFor, prisma, userData.languages, "view");
            view = await prisma.view.create({
                data: {
                    by: { connect: { id: userData.id } },
                    name: labels[0],
                    ...createMapper[input.viewFor](objectToView),
                },
            });
        }
        let isOwn = false;
        switch (input.viewFor) {
            case ViewFor.Organization:
                const roles = await OrganizationModel.query.hasRole(prisma, userData.id, [input.forId]);
                isOwn = Boolean(roles[0]);
                break;
            case ViewFor.Api:
            case ViewFor.ApiVersion:
            case ViewFor.Note:
            case ViewFor.NoteVersion:
            case ViewFor.Project:
            case ViewFor.ProjectVersion:
            case ViewFor.Routine:
            case ViewFor.RoutineVersion:
            case ViewFor.SmartContract:
            case ViewFor.SmartContractVersion:
            case ViewFor.Standard:
            case ViewFor.StandardVersion:
                const { delegate: rootObjectDelegate } = getLogic(["delegate"], input.viewFor.replace("Version", ""), userData.languages, "ViewModel.view2");
                const rootObject = await rootObjectDelegate(prisma).findFirst({
                    where: {
                        AND: [
                            { id: dataMapper[input.viewFor](objectToView).id },
                            {
                                OR: [
                                    OrganizationModel.query.isMemberOfOrganizationQuery(userData.id),
                                    { ownedByUser: { id: userData.id } },
                                ],
                            },
                        ],
                    },
                });
                if (rootObject)
                    isOwn = true;
                break;
            case ViewFor.Question:
                const question = await prisma.question.findFirst({ where: { id: input.forId, createdBy: { id: userData.id } } });
                if (question)
                    isOwn = true;
                break;
            case ViewFor.User:
                isOwn = userData.id === input.forId;
                break;
        }
        if (isOwn)
            return true;
        const redisKey = `view:${userData.id}_${dataMapper[input.viewFor](objectToView).id}_${input.viewFor}`;
        const client = await initializeRedis();
        const lastViewed = await client.get(redisKey);
        if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
            const { delegate: rootObjectDelegate } = getLogic(["delegate"], input.viewFor.replace("Version", ""), userData.languages, "ViewModel.view3");
            await rootObjectDelegate(prisma).update({
                where: { id: input.forId },
                data: { views: dataMapper[input.viewFor](objectToView).views + 1 },
            });
        }
        await client.set(redisKey, new Date().toISOString());
        return true;
    },
    deleteViews,
    clearViews,
});
//# sourceMappingURL=view.js.map