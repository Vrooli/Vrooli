import { transferValidation } from "@local/validation";
import { ApiModel, NoteModel, OrganizationModel, ProjectModel, RoutineModel, SmartContractModel, StandardModel } from ".";
import { noNull, permissionsSelectHelper, selPad } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { Notify } from "../notify";
import { getSingleTypePermissions, isOwnerAdminCheck } from "../validators";
const __typename = "Transfer";
const suppFields = [];
export const TransferableFieldMap = {
    Api: "api",
    Note: "note",
    Project: "project",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
};
export const transfer = (prisma) => ({
    checkTransferRequests: async (owners, userData) => {
        const orgIds = owners.filter(o => o.__typename === "Organization").map(o => o.id);
        const isAdmins = await OrganizationModel.query.hasRole(prisma, userData.id, orgIds);
        const requiresTransferRequest = owners.map((o, i) => {
            if (o.__typename === "User")
                return o.id !== userData.id;
            const orgIdIndex = orgIds.indexOf(o.id);
            return !isAdmins[orgIdIndex];
        });
        return requiresTransferRequest;
    },
    requestSend: async (input, userData) => {
        const object = { __typename: input.objectType, id: input.objectConnect };
        const { delegate, validate } = getLogic(["delegate", "validate"], object.__typename, userData.languages, "Transfer.request-object");
        const permissionData = await delegate(prisma).findUnique({
            where: { id: object.id },
            select: validate.permissionsSelect,
        });
        const owner = permissionData && validate.owner(permissionData, userData.id);
        if (!owner || !isOwnerAdminCheck(owner, userData.id))
            throw new CustomError("0286", "NotAuthorizedToTransfer", userData.languages);
        const toType = input.toOrganizationConnect ? "Organization" : "User";
        const toId = input.toOrganizationConnect || input.toUserConnect;
        const request = await prisma.transfer.create({
            data: {
                fromUser: owner.User ? { connect: { id: owner.User.id } } : undefined,
                fromOrganization: owner.Organization ? { connect: { id: owner.Organization.id } } : undefined,
                [TransferableFieldMap[object.__typename]]: { connect: { id: object.id } },
                toUser: toType === "User" ? { connect: { id: toId } } : undefined,
                toOrganization: toType === "Organization" ? { connect: { id: toId } } : undefined,
                status: "Pending",
                message: input.message,
            },
            select: { id: true },
        });
        const pushNotification = Notify(prisma, userData.languages).pushTransferRequestSend(request.id, object.__typename, object.id);
        if (toType === "User")
            await pushNotification.toUser(toId);
        else
            await pushNotification.toOrganization(toId);
        return request.id;
    },
    requestReceive: async (info, input, userData) => {
        return "";
    },
    cancel: async (transferId, userData) => {
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
            select: {
                id: true,
                fromOrganizationId: true,
                fromUserId: true,
                toOrganizationId: true,
                toUserId: true,
                status: true,
            },
        });
        if (!transfer)
            throw new CustomError("0293", "TransferNotFound", userData.languages);
        if (transfer.status === "Accepted")
            throw new CustomError("0294", "TransferAlreadyAccepted", userData.languages);
        if (transfer.status === "Denied")
            throw new CustomError("0295", "TransferAlreadyRejected", userData.languages);
        if (transfer.fromOrganizationId) {
            const { validate } = getLogic(["validate"], "Organization", userData.languages, "Transfer.cancel");
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.fromOrganizationId },
                select: permissionsSelectHelper(validate.permissionsSelect, userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validate.owner(permissionData, userData.id), userData.id))
                throw new CustomError("0300", "TransferRejectNotAuthorized", userData.languages);
        }
        else if (transfer.fromUserId !== userData.id) {
            throw new CustomError("0301", "TransferRejectNotAuthorized", userData.languages);
        }
        await prisma.transfer.delete({
            where: { id: transferId },
        });
    },
    accept: async (transferId, userData) => {
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
        });
        if (!transfer)
            throw new CustomError("0287", "TransferNotFound", userData.languages);
        if (transfer.status === "Accepted")
            throw new CustomError("0288", "TransferAlreadyAccepted", userData.languages);
        if (transfer.status === "Denied")
            throw new CustomError("0289", "TransferAlreadyRejected", userData.languages);
        if (transfer.toOrganizationId) {
            const { validate } = getLogic(["validate"], "Organization", userData.languages, "Transfer.accept");
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: permissionsSelectHelper(validate.permissionsSelect, userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validate.owner(permissionData, userData.id), userData.id))
                throw new CustomError("0302", "TransferAcceptNotAuthorized", userData.languages);
        }
        else if (transfer.toUserId !== userData.id) {
            throw new CustomError("0303", "TransferAcceptNotAuthorized", userData.languages);
        }
        const typeField = ["apiId", "noteId", "projectId", "routineId", "smartContractId", "standardId"].find((field) => transfer[field] !== null);
        if (!typeField)
            throw new CustomError("0290", "TransferMissingData", userData.languages);
        const type = typeField.replace("Id", "");
        await prisma.transfer.update({
            where: { id: transferId },
            data: {
                status: "Accepted",
                [type]: {
                    update: {
                        ownerId: transfer.toUserId,
                        organizationId: transfer.toOrganizationId,
                    },
                },
            },
        });
    },
    deny: async (transferId, userData, reason) => {
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
        });
        if (!transfer)
            throw new CustomError("0320", "TransferNotFound", userData.languages);
        if (transfer.status === "Accepted")
            throw new CustomError("0291", "TransferAlreadyAccepted", userData.languages);
        if (transfer.status === "Denied")
            throw new CustomError("0292", "TransferAlreadyRejected", userData.languages);
        if (transfer.toOrganizationId) {
            const { validate } = getLogic(["validate"], "Organization", userData.languages, "Transfer.reject");
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: permissionsSelectHelper(validate.permissionsSelect, userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validate.owner(permissionData, userData.id), userData.id))
                throw new CustomError("0312", "TransferRejectNotAuthorized", userData.languages);
        }
        else if (transfer.toUserId !== userData.id) {
            throw new CustomError("0313", "TransferRejectNotAuthorized", userData.languages);
        }
        const typeField = ["apiId", "noteId", "projectId", "routineId", "smartContractId", "standardId"].find((field) => transfer[field] !== null);
        if (!typeField)
            throw new CustomError("0290", "TransferMissingData", userData.languages);
        const type = typeField.replace("Id", "");
        await prisma.transfer.update({
            where: { id: transferId },
            data: {
                status: "Denied",
                denyReason: reason,
            },
        });
    },
});
export const TransferModel = ({
    __typename,
    delegate: (prisma) => prisma.transfer,
    display: {
        select: () => ({
            id: true,
            api: selPad(ApiModel.display.select),
            note: selPad(NoteModel.display.select),
            project: selPad(ProjectModel.display.select),
            routine: selPad(RoutineModel.display.select),
            smartContract: selPad(SmartContractModel.display.select),
            standard: selPad(StandardModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api)
                return ApiModel.display.label(select.api, languages);
            if (select.note)
                return NoteModel.display.label(select.note, languages);
            if (select.project)
                return ProjectModel.display.label(select.project, languages);
            if (select.routine)
                return RoutineModel.display.label(select.routine, languages);
            if (select.smartContract)
                return SmartContractModel.display.label(select.smartContract, languages);
            if (select.standard)
                return StandardModel.display.label(select.standard, languages);
            return "";
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            fromOwner: {
                fromUser: "User",
                fromOrganization: "Organization",
            },
            object: {
                api: "Api",
                note: "Note",
                project: "Project",
                routine: "Routine",
                smartContract: "SmartContract",
                standard: "Standard",
            },
            toOwner: {
                toUser: "User",
                toOrganization: "Organization",
            },
        },
        prismaRelMap: {
            __typename,
            fromUser: "User",
            fromOrganization: "Organization",
            toUser: "User",
            toOrganization: "Organization",
            api: "Api",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            update: async ({ data }) => ({
                message: noNull(data.message),
            }),
        },
        yup: transferValidation,
    },
    transfer,
    validate: {},
});
//# sourceMappingURL=transfer.js.map