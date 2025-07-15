import { ApiVersionConfig, type ApiVersionConfigObject, type BaseConfigObject, CodeVersionConfig, type CodeVersionConfigObject, ModelType, NoteVersionConfig, type NoteVersionConfigObject, type Owner, ProjectVersionConfig, type ProjectVersionConfigObject, type Resource, type ResourceShape, ResourceType, type ResourceVersionShape, RoutineVersionConfig, type RoutineVersionConfigObject, type SessionUser, StandardVersionConfig, type StandardVersionConfigObject, mergeDeep, shapeResource } from "@vrooli/shared";
import { createHash } from "crypto";
import jwt from "jsonwebtoken";
import { createOneHelper } from "../actions/creates.js";
import { updateOneHelper } from "../actions/updates.js";
import { type RequestService } from "../auth/request.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { getAuthenticatedData } from "../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../validators/permissions.js";
import { combineQueries } from "./combineQueries.js";
import { InfoConverter } from "./infoConverter.js";
import { permissionsSelectHelper } from "./permissionsSelectHelper.js";
import { type PartialApiInfo } from "./types.js";

/** The current overall export format version. */
const EXPORT_VERSION = "1.0.0";

type ImportDataBase<Typename extends `${ModelType}`> = {
    /** The object's export format version. */
    __version: string;
    /** The object's type name. */
    __typename: Typename;
    /** The object's shape. */
    shape: unknown;
}

type ShapeBase<Shape extends object, OmitFields extends string> = Omit<Shape, OmitFields | "__typename">;

export type ResourceImportData = ImportDataBase<"Resource"> & {
    shape: ShapeBase<ResourceShape, "owner" | "versions"> & {
        versions: ResourceVersionImportData[];
    }
}

export type ResourceVersionImportData = ImportDataBase<"ResourceVersion"> & {
    shape: ShapeBase<ResourceVersionShape, "config"> & {
        config: ApiVersionConfigObject | CodeVersionConfigObject | NoteVersionConfigObject | ProjectVersionConfigObject | RoutineVersionConfigObject | StandardVersionConfigObject;
    }
}

/** The list of objects being imported or exported. */
type ImportObjects = ResourceImportData | ResourceVersionImportData;

/** The overall import/export data. */
export type ImportData = {
    /** 
     * The date and time the export was created (as an ISO 8601 string). 
     * Example: "2021-01-01T00:00:00.000Z"
     */
    __exportedAt: string;
    /**
     * Signature that can be used to verify that the export data came from this application. 
     * It's a signed hash of the export data.
     * 
     * NOTE: This uses the same signing mechanism as the JWT tokens.
     */
    __signature: string;
    /** 
     * The source of the export (i.e. the name of the application that created the export). 
     * 
     * NOTE: This field is not verified.
     */
    __source: string;
    /** The overall export format version. */
    __version: string;
    /** The list of objects being imported or exported. */
    data: ImportObjects[];
}

/**
 * Configuration for importing data.
 */
export type ImportConfig = {
    /** Whether to allow importing data from a different source. */
    allowForeignData: boolean;
    /**
     * Who to assign ownership to.
     * 
     * Can be a user or a team.
     */
    assignObjectsTo: Pick<Owner, "__typename" | "id">;
    /** Behavior when an object already exists in the database. */
    onConflict: "overwrite" | "skip" | "error";
    /** 
     * Whether we should skip permission checks if overwriting an object.
     * 
     * WARNING: Only use this for seeding/testing.
     */
    skipPermissions?: boolean;
    /**
     * The user requesting the import/export.
     */
    userData: Pick<SessionUser, "id" | "languages">;
    /** If this import is part of seeding */
    isSeeding: boolean;
}

/**
 * The result of importing data.
 */
type ImportResult = {
    imported: number;
    skipped: number;
    errors: number;
}

/**
 * Behavior after the export is complete.
 */
enum BehaviorAfterExport {
    Anonymize = "Anonymize",
    Delete = "Delete",
    NoAction = "NoAction",
}

/**
 * Base configuration for exporting data.
 */
type ExportConfigBase = {
    /** What to do after exporting the object. */
    afterExport: BehaviorAfterExport;
    /** Whether the data should be downloadable. */
    downloadable: boolean;
}

/**
 * Configuration for exporting a single object.
 */
type ExportConfigSingleObject = ExportConfigBase & {
    __type: "SingleObject";
    /** The object type to export. */
    objectType: ModelType;
    /** The object ID to export. */
    objectId: string;
}

/**
 * Configuration for exporting user data.
 */
type ExportConfigUser = ExportConfigBase & {
    __type: "User";
    /** The user ID to export. */
    userId: string;
    /** What data should be exported */
    flags: {
        /** If true, will export all data - regardless of other fields selected. */
        all: boolean;
        /** Export account data (profile, emails, wallets, awards, focus modes, payment history, api usage, user stats) */
        account?: boolean;
        /** Export api data? */
        apis?: boolean;
        /** Export bookmarks data? */
        bookmarks?: boolean;
        /** Export bots data? */
        bots?: boolean;
        /** Export chats data? */
        chats?: boolean;
        /** Export codes data? */
        codes?: boolean;
        /** Export comments data? */
        comments?: boolean;
        /** Export issues data? */
        issues?: boolean;
        /** Export notes data? */
        notes?: boolean;
        /** Export pull requests data? */
        pullRequests?: boolean;
        /** Export projects data? */
        projects?: boolean;
        /** Export reactions data? */
        reactions?: boolean;
        /** Export reminders data? */
        reminders?: boolean;
        /** Export reports data? */
        reports?: boolean;
        /** Export routines data? */
        routines?: boolean;
        /** Export runs data? */
        runs?: boolean;
        /** Export schedules data? */
        schedules?: boolean;
        /** Export standards data? */
        standards?: boolean;
        /** Export teams data? */
        teams?: boolean;
        /** Export views data? */
        views?: boolean;
    }
}

/**
 * Configuration for exporting team data.
 */
type ExportConfigTeam = Omit<ExportConfigUser, "__type" | "userId"> & {
    __type: "Team";
    /** The team ID to export. */
    teamId: string;
}

export type ExportConfig = ExportConfigSingleObject | ExportConfigUser | ExportConfigTeam;

/**
 * Abstract class for importing and exporting data.
 * 
 * Each importable/exportable object must extend this class.
 */
abstract class AbstractImportExport<Import extends ImportDataBase<`${ModelType}`>, DbModel extends object> {
    /** Builds a minimal request object for importing/exporting data. */
    protected buildRequest(config: ImportConfig): Parameters<typeof RequestService.assertRequestFrom>[0] {
        return {
            session: {
                apiToken: undefined,
                fromSafeOrigin: true,
                isLoggedIn: true,
                languages: config.userData.languages,
                users: [config.userData],
            },
        };
    }

    public abstract getInfoCreate(): Promise<PartialApiInfo>;

    public abstract getInfoUpdate(): Promise<PartialApiInfo>;

    /** Imports new data into the system. */
    public abstract importCreate(data: Import, config: ImportConfig): Promise<DbModel>;

    /** Imports existing data into the system. */
    public abstract importUpdate(existing: DbModel, data: Import, config: ImportConfig): Promise<DbModel>;

    /** Exports data from the system. */
    public abstract export(record: DbModel, config: ExportConfig): Promise<Import>;
}

export class ResourceImportExport extends AbstractImportExport<ResourceImportData, Resource> {
    public async getInfoCreate() {
        const info = (await import("../endpoints/generated/resource_createOne.js")).resource_createOne;
        return info;
    }

    public async getInfoUpdate() {
        const info = (await import("../endpoints/generated/resource_updateOne.js")).resource_updateOne;
        return info;
    }

    private shapeData({ shape }: Pick<ResourceImportData, "shape">, { assignObjectsTo }: Pick<ImportConfig, "assignObjectsTo">): ResourceShape {
        const versions = shape.versions.map(version => {
            // Return without "root" field
            const { codeLanguage, config, resourceSubType, root, ...rest } = version.shape;
            let configJson: BaseConfigObject;
            switch (root?.resourceType) {
                case ResourceType.Api:
                    configJson = new ApiVersionConfig({ config: config as ApiVersionConfigObject }).export();
                    break;
                case ResourceType.Code:
                    configJson = new CodeVersionConfig({ codeLanguage, config: config as CodeVersionConfigObject }).export();
                    break;
                case ResourceType.Note:
                    configJson = new NoteVersionConfig({ config: config as NoteVersionConfigObject }).export();
                    break;
                case ResourceType.Project:
                    configJson = new ProjectVersionConfig({ config: config as ProjectVersionConfigObject }).export();
                    break;
                case ResourceType.Routine:
                    configJson = new RoutineVersionConfig({ config: config as RoutineVersionConfigObject, resourceSubType: resourceSubType! }).export();
                    break;
                case ResourceType.Standard:
                    configJson = new StandardVersionConfig({ config: config as StandardVersionConfigObject, resourceSubType: resourceSubType! }).export();
                    break;
                default:
                    configJson = { __version: "0.0.0" } as BaseConfigObject;
                    break;
            }
            // Serialize the config
            const codeVersionShape = {
                ...rest,
                __typename: "ResourceVersion" as const,
                codeLanguage,
                config: configJson,
                resourceSubType,
            };
            return codeVersionShape;
        });
        const resourceShape = {
            ...shape,
            __typename: "Resource" as const,
            owner: assignObjectsTo,
            versions,
        };
        return resourceShape;
    }

    public async importCreate(data: ResourceImportData, config: ImportConfig): Promise<Resource> {
        const info = await this.getInfoCreate();
        const resourceShape = this.shapeData(data, config);
        const input = shapeResource.create(resourceShape);
        const req = this.buildRequest(config);
        const adminFlags = config.isSeeding ? { isSeeding: true } : undefined;
        const result = await createOneHelper({ adminFlags, info, input, objectType: "Resource", req });
        return result as Resource;
    }

    public async importUpdate(existing: Resource, data: ResourceImportData, config: ImportConfig): Promise<Resource> {
        const info = await this.getInfoUpdate();
        const resourceShape = this.shapeData(data, config);
        const input = shapeResource.update({ ...existing, owner: existing.owner ?? null }, resourceShape);
        const req = this.buildRequest(config);
        const adminFlags = config.isSeeding ? { isSeeding: true } : undefined;
        const result = await updateOneHelper({ adminFlags, info, input, objectType: "Resource", req });
        return result as Resource;
    }

    public async export(record: Resource): Promise<ResourceImportData> {
        //TODO
        return {} as any;
    }
}

/**
 * Converts stringified JSON data into a list of imports.
 */
export function parseImportData(data: string): ImportData {
    let parsedData: ImportData;
    try {
        parsedData = JSON.parse(data);
    } catch (error) {
        throw new CustomError("0461", "InvalidArgs");
    }
    // Make sure the data is appropriately formatted
    if (
        !parsedData ||
        typeof parsedData !== "object" ||
        !Object.prototype.hasOwnProperty.call(parsedData, "__version") ||
        typeof parsedData.__version !== "string" ||
        !Object.prototype.hasOwnProperty.call(parsedData, "data") ||
        !Array.isArray(parsedData.data) ||
        parsedData.data.some((item) => !Object.prototype.hasOwnProperty.call(item, "__typename") || !Object.prototype.hasOwnProperty.call(item, "shape"))
    ) {
        throw new CustomError("0462", "InvalidArgs", { data: JSON.stringify(parsedData).slice(0, 100) });
    }

    return parsedData;
}

/**
 * Creates a signature for the export data.
 * Used to verify that the export data came from this application.
 * 
 * NOTE: We stringify and hash the data before signing it to reduce the signature size.
 * 
 * @param data The data being imported/exported.
 * @param owner The owner of the data.
 * @returns The signature.
 */
export function createExportSignature(data: ImportObjects[], owner: Pick<Owner, "__typename" | "id">): string {
    // Stringify the data and owner
    const stringified = JSON.stringify({ data, owner });
    // Hash the stringified data in a cryptographically secure way
    const hash = createHash("sha256");
    hash.update(stringified);
    const hashed = hash.digest("hex");
    // Sign the hashed data with our private key using JWT (with no expiration)
    const signed = jwt.sign(hashed, process.env.JWT_PRIV ?? "", { algorithm: "RS256" });
    // Return the signed data
    return signed;
}

/**
 * Imports data into the database.
 * 
 * @param data The parsed import data.
 * @param config The import configuration.
 * @returns The number of objects imported/skipped/errored.
 */
export async function importData(data: ImportData, config: ImportConfig): Promise<ImportResult> {
    const result: ImportResult = {
        imported: 0,
        skipped: 0,
        errors: 0,
    };

    // If `allowForeignData` is false, check that the source of the export matches the current application
    if (config.allowForeignData === false) {
        const expectedSignature = createExportSignature(data.data, config.assignObjectsTo);
        if (data.__signature !== expectedSignature) {
            throw new CustomError("0584", "InvalidArgs");
        }
    }

    // Check that the version is supported
    if (data.__version !== EXPORT_VERSION) {
        throw new CustomError("0585", "InvalidArgs", { version: data.__version.slice(0, 10) });
    }

    // If the userData is not the same as assignObjectTo, check that the user has permission to import
    if (config.skipPermissions !== true || config.assignObjectsTo.__typename !== "User" || config.assignObjectsTo.id !== config.userData.id) {
        const objectType = config.assignObjectsTo.__typename;
        const id = config.assignObjectsTo.id;
        const authDataById = await getAuthenticatedData({ [objectType]: [id] }, config.userData);
        if (Object.keys(authDataById).length === 0) {
            throw new CustomError("0586", "NotFound", { id, objectType });
        }
        // If you can delete, you can import
        await permissionsCheck(authDataById, { ["Delete"]: [id as string] }, {}, config.userData);
    }

    // Group objects by their __typename, so that we can efficiently batch operations
    const groupedData: { [objectType: string]: ImportObjects[] } = {};
    for (const obj of data.data) {
        const objectType = obj.__typename;
        if (!groupedData[objectType]) {
            groupedData[objectType] = [];
        }
        groupedData[objectType].push(obj);
    }

    // Process each object type group.
    for (const [objectType, objects] of Object.entries(groupedData)) {
        // Determine the importer instance based on objectType
        let importer: AbstractImportExport<ImportDataBase<`${ModelType}`>, object> | null = null;
        if (objectType === ModelType.Resource) {
            importer = new ResourceImportExport();
        } else {
            // Unsupported object type
            result.errors += objects.length;
            continue;
        }

        const { dbTable, format, validate } = ModelMap.getLogic(["dbTable", "format", "validate"], objectType as `${ModelType}`);


        // Objects with publicIds can be updated during seeding. Everything else will be created
        const publicIdsToCheck = config.isSeeding ? objects
            .map(obj => obj.shape["publicId"])
            .filter(publicId => publicId != null)
            : [];

        // Select using all fields required for checking permissions, 
        // combined with the shape that is imported/exported.
        // This ensures that we can check permissions, while being able to correctly 
        // determine what relations are being created/updated/deleted.
        const selectPermissions = permissionsSelectHelper(validate().permissionsSelect, config.userData.id);
        const partialInfoCreate = InfoConverter.get().fromApiToPartialApi(await importer.getInfoCreate(), format.apiRelMap) as PartialApiInfo;
        const selectImportCreate = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfoCreate)?.select;
        const partialInfoUpdate = InfoConverter.get().fromApiToPartialApi(await importer.getInfoUpdate(), format.apiRelMap) as PartialApiInfo;
        const selectImportUpdate = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfoUpdate)?.select;
        const combinedSelect = combineQueries([selectPermissions, selectImportCreate, selectImportUpdate], { mergeMode: "loose" });
        // Do a single findMany query for all objects of this type that have a publicId.
        const existingObjectsPrisma = publicIdsToCheck.length > 0
            ? await DbProvider.get()[dbTable].findMany({
                where: { publicId: { in: publicIdsToCheck } },
                select: { ...combinedSelect, id: true, publicId: true }, // Make sure ID and publicId are included
            })
            : [];
        const partialInfo = mergeDeep(partialInfoCreate, partialInfoUpdate);
        const existingObjects = existingObjectsPrisma.map((obj) => {
            return InfoConverter.get().fromDbToApi(obj, partialInfo);
        });

        // Create a lookup map for existing objects by their publicId
        const existingMap = new Map<string, object>();
        for (const existing of existingObjects) {
            existingMap.set(existing["publicId"], existing);
        }

        // Process each object in the group.
        for (const obj of objects) {
            const publicId = obj.shape["publicId"];
            const canSkipPermissions = config.onConflict === "overwrite" && config.skipPermissions === true;
            let canImport = canSkipPermissions || !publicId;

            // If the object has a publicId, it may already exist.
            if (publicId) {
                const existing = existingMap.get(publicId);
                if (!existing) {
                    canImport = true;
                } else if (config.onConflict === "skip") {
                    result.skipped++;
                    continue;
                } else if (config.onConflict === "error") {
                    result.errors++;
                    continue;
                } else {
                    // Check that the user has permission to overwrite (i.e. delete) the object
                    try {
                        // Permissions use ID instead of publicId
                        const id = obj["id"];
                        if (!id) {
                            canImport = false;
                        } else {
                            await permissionsCheck(
                                { [id]: { __typename: objectType as `${ModelType}`, ...existing } },
                                { ["Delete"]: [id] },
                                {},
                                config.userData,
                            );
                            canImport = true;
                        }
                    } catch (error) {
                        result.errors++;
                        continue;
                    }
                }
            }

            if (canImport) {
                const existing = publicId ? existingMap.get(publicId) : null;
                if (existing) {
                    await importer.importUpdate(existing, obj, config);
                } else {
                    await importer.importCreate(obj, config);
                }
                result.imported++;
            } else {
                result.skipped++;
            }
        }
    }

    //TODO send notification/email/something

    return result;
}

//TODO add export function. Should upload file(s) (multiple files might be created for large exports) to S3 with expiration date, and send email with download link.
//TODO import/export order matters. Some objects depend on others. For example, should always import tags first, and export them last.
