import { Code, CodeShape, CodeVersionConfig, CodeVersionConfigObject, CodeVersionShape, ModelType, Owner, Routine, RoutineShape, RoutineVersionConfig, RoutineVersionConfigObject, RoutineVersionShape, SessionUser, Standard, StandardShape, StandardVersionShape, shapeCode, shapeRoutine, shapeStandard } from "@local/shared";
import { createHash } from "crypto";
import jwt from "jsonwebtoken";
import { createOneHelper } from "../actions/creates.js";
import { updateOneHelper } from "../actions/updates.js";
import { RequestService } from "../auth/request.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { getAuthenticatedData } from "../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../validators/permissions.js";
import { permissionsSelectHelper } from "./permissionsSelectHelper.js";

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

export type CodeImportData = ImportDataBase<"Code"> & {
    shape: ShapeBase<CodeShape, "owner" | "versions"> & {
        versions: CodeVersionImportData[];
    }
}

export type CodeVersionImportData = ImportDataBase<"CodeVersion"> & {
    shape: ShapeBase<CodeVersionShape, "data"> & {
        data: CodeVersionConfigObject;
    }
}

export type RoutineImportData = ImportDataBase<"Routine"> & {
    shape: ShapeBase<RoutineShape, "owner" | "versions"> & {
        versions: RoutineVersionImportData[];
    }
}

export type RoutineVersionImportData = ImportDataBase<"RoutineVersion"> & {
    shape: ShapeBase<RoutineVersionShape, "config"> & {
        config: RoutineVersionConfigObject;
    }
}

export type StandardImportData = ImportDataBase<"Standard"> & {
    shape: ShapeBase<StandardShape, "owner" | "versions"> & {
        versions: StandardVersionImportData[];
    }
}

export type StandardVersionImportData = ImportDataBase<"StandardVersion"> & {
    shape: ShapeBase<StandardVersionShape, "">;
}

/** The list of objects being imported or exported. */
type ImportObjects = CodeImportData | CodeVersionImportData | RoutineImportData | RoutineVersionImportData | StandardImportData | StandardVersionImportData;

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
        /** Export questions data? */
        questions?: boolean;
        /** Export question answers data? */
        questionAnswers?: boolean;
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

    /** Imports new data into the system. */
    public abstract importCreate(data: Import, config: ImportConfig): Promise<DbModel>;

    /** Imports existing data into the system. */
    public abstract importUpdate(existing: DbModel, data: Import, config: ImportConfig): Promise<DbModel>;

    /** Exports data from the system. */
    public abstract export(record: DbModel, config: ExportConfig): Promise<Import>;
}

class CodeImportExport extends AbstractImportExport<CodeImportData, Code> {
    private shapeData({ shape }: Pick<CodeImportData, "shape">, { assignObjectsTo }: Pick<ImportConfig, "assignObjectsTo">): CodeShape {
        const versions = shape.versions.map(version => {
            // Return without "root" field
            const { codeLanguage, content, data, root: _root, ...rest } = version.shape;
            // Serialize the config
            const codeVersionShape = {
                ...rest,
                __typename: "CodeVersion" as const,
                codeLanguage,
                content,
                data: new CodeVersionConfig({ content, codeLanguage, data }).serialize("json"),
            };
            return codeVersionShape;
        });
        const codeShape = {
            ...shape,
            __typename: "Code" as const,
            owner: assignObjectsTo,
            versions,
        };
        return codeShape;
    }

    public async importCreate(data: CodeImportData, config: ImportConfig): Promise<Code> {
        const info = (await import("../endpoints/generated/code_createOne.js")).code_createOne;
        const codeShape = this.shapeData(data, config);
        const input = shapeCode.create(codeShape);
        const req = this.buildRequest(config);
        console.log("creating imported code", JSON.stringify(input, null, 2));
        const result = await createOneHelper({ info, input, objectType: "Code", req });
        return result as Code;
    }

    public async importUpdate(existing: Code, data: CodeImportData, config: ImportConfig): Promise<Code> {
        const info = (await import("../endpoints/generated/code_updateOne.js")).code_updateOne;
        const codeShape = this.shapeData(data, config);
        const input = shapeCode.update({ ...existing, owner: existing.owner ?? null }, codeShape);
        const req = this.buildRequest(config);
        console.log("updating imported code", JSON.stringify(input, null, 2));
        const result = await updateOneHelper({ info, input, objectType: "Code", req });
        return result as Code;
    }

    public async export(record: Code): Promise<CodeImportData> {
        //TODO
        return {} as any;
    }
}

class RoutineImportExport extends AbstractImportExport<RoutineImportData, Routine> {
    private shapeData({ shape }: Pick<RoutineImportData, "shape">, { assignObjectsTo }: Pick<ImportConfig, "assignObjectsTo">): RoutineShape {
        const versions = shape.versions.map(version => {
            // Return without "root" field
            const { config, root: _root, ...rest } = version.shape;
            // Serialize the config
            const routineVersionShape = {
                ...rest,
                __typename: "RoutineVersion" as const,
                config: new RoutineVersionConfig(config).serialize("json"),
            };
            return routineVersionShape;
        });
        const routineShape = {
            ...shape,
            __typename: "Routine" as const,
            owner: assignObjectsTo,
            versions,
        };
        return routineShape;
    }

    public async importCreate(data: RoutineImportData, config: ImportConfig): Promise<Routine> {
        const info = (await import("../endpoints/generated/routine_createOne.js")).routine_createOne;
        const routineShape = this.shapeData(data, config);
        const input = shapeRoutine.create(routineShape);
        const req = this.buildRequest(config);
        console.log("creating imported routine", JSON.stringify(input, null, 2));
        const result = await createOneHelper({ info, input, objectType: "Routine", req });
        return result as Routine;
    }

    public async importUpdate(existing: Routine, data: RoutineImportData, config: ImportConfig): Promise<Routine> {
        const info = (await import("../endpoints/generated/routine_updateOne.js")).routine_updateOne;
        const routineShape = this.shapeData(data, config);
        const input = shapeRoutine.update({ ...existing, owner: existing.owner ?? null }, routineShape);
        const req = this.buildRequest(config);
        console.log("updating imported routine", JSON.stringify(input, null, 2));
        const result = await updateOneHelper({ info, input, objectType: "Routine", req });
        return result as Routine;
    }

    public async export(record: Routine): Promise<RoutineImportData> {
        //TODO
        return {} as any;
    }
}

class StandardImportExport extends AbstractImportExport<StandardImportData, Standard> {
    private shapeData({ shape }: Pick<StandardImportData, "shape">, { assignObjectsTo }: Pick<ImportConfig, "assignObjectsTo">): StandardShape {
        const versions = shape.versions.map(version => {
            // Return without "root" field
            const { root: _root, ...rest } = version.shape;
            // Serialize the config
            const standardVersionShape = {
                ...rest,
                __typename: "StandardVersion" as const,
            };
            return standardVersionShape;
        });
        const standardShape = {
            ...shape,
            __typename: "Standard" as const,
            owner: assignObjectsTo,
            versions,
        };
        return standardShape;
    }

    public async importCreate(data: StandardImportData, config: ImportConfig): Promise<Standard> {
        const info = (await import("../endpoints/generated/standard_createOne.js")).standard_createOne;
        const standardShape = this.shapeData(data, config);
        const input = shapeStandard.create(standardShape);
        const req = this.buildRequest(config);
        console.log("creating imported standard", JSON.stringify(input, null, 2));
        const result = await createOneHelper({ info, input, objectType: "Standard", req });
        return result as Standard;
    }

    public async importUpdate(existing: Standard, data: StandardImportData, config: ImportConfig): Promise<Standard> {
        const info = (await import("../endpoints/generated/standard_updateOne.js")).standard_updateOne;
        const standardShape = this.shapeData(data, config);
        const input = shapeStandard.update({ ...existing, owner: existing.owner ?? null }, standardShape);
        const req = this.buildRequest(config);
        console.log("updating imported standard", JSON.stringify(input, null, 2));
        const result = await updateOneHelper({ info, input, objectType: "Standard", req });
        return result as Standard;
    }

    public async export(record: Standard): Promise<StandardImportData> {
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
        if (objectType === ModelType.Code) {
            importer = new CodeImportExport();
        } else if (objectType === ModelType.Routine) {
            importer = new RoutineImportExport();
        } else if (objectType === ModelType.Standard) {
            importer = new StandardImportExport();
        } else {
            // Unsupported object type
            result.errors += objects.length;
            continue;
        }

        const { dbTable, idField, validate } = ModelMap.getLogic(["dbTable", "idField", "validate"], objectType as `${ModelType}`);

        // Gather all non-null IDs from this group.
        const idsToCheck = objects
            .map(obj => obj.shape[idField])
            .filter(id => id != null);

        // Do a single findMany query for all objects of this type that have an ID.
        const existingObjects = idsToCheck.length > 0
            ? await DbProvider.get()[dbTable].findMany({
                where: { [idField]: { in: idsToCheck } },
                select: permissionsSelectHelper(validate().permissionsSelect, config.userData.id),
            })
            : [];

        // Create a lookup map for existing objects by their ID.
        const existingMap = new Map<string, object>();
        for (const existing of existingObjects) {
            existingMap.set(existing[idField], existing);
        }

        // Process each object in the group.
        for (const obj of objects) {
            const objId = obj.shape[idField];
            const canSkipPermissions = config.onConflict === "overwrite" && config.skipPermissions === true;
            let canImport = canSkipPermissions || !objId;

            // If the object has an ID, it may already exist.
            if (objId) {
                const existing = existingMap.get(objId);
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
                        await permissionsCheck(
                            { [objId]: { __typename: objectType as `${ModelType}`, ...existing } },
                            { ["Delete"]: [objId] },
                            {},
                            config.userData,
                        );
                        canImport = true;
                    } catch (error) {
                        result.errors++;
                        continue;
                    }
                }
            }

            if (canImport) {
                const existing = existingMap.get(objId);
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
