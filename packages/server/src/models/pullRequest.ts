import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { ApiModel } from "./api";
import { ApiVersionModel } from "./apiVersion";
import { NoteModel } from "./note";
import { NoteVersionModel } from "./noteVersion";
import { ProjectModel } from "./project";
import { ProjectVersionModel } from "./projectVersion";
import { RoutineModel } from "./routine";
import { RoutineVersionModel } from "./routineVersion";
import { SmartContractModel } from "./smartContract";
import { SmartContractVersionModel } from "./smartContractVersion";
import { StandardModel } from "./standard";
import { StandardVersionModel } from "./standardVersion";
import { Displayer } from "./types";

const __typename = 'PullRequest' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.pull_requestSelect,
    Prisma.pull_requestGetPayload<SelectWrap<Prisma.pull_requestSelect>>
> => ({
    select: () => ({
        id: true,
        toApi: { select: ApiModel.display.select() },
        fromApiVersion: { select: ApiVersionModel.display.select() },
        toNote: { select: NoteModel.display.select() },
        fromNoteVersion: { select: NoteVersionModel.display.select() },
        toProject: { select: ProjectModel.display.select() },
        fromProjectVersion: { select: ProjectVersionModel.display.select() },
        toRoutine: { select: RoutineModel.display.select() },
        fromRoutineVersion: { select: RoutineVersionModel.display.select() },
        toSmartContract: { select: SmartContractModel.display.select() },
        fromSmartContractVersion: { select: SmartContractVersionModel.display.select() },
        toStandard: { select: StandardModel.display.select() },
        fromStandardVersion: { select: StandardVersionModel.display.select() },
    }),
    // Label is from -> to
    label: (select, languages) => {
        const from = select.fromApiVersion ? ApiVersionModel.display.label(select.fromApiVersion as any, languages) :
            select.fromNoteVersion ? NoteVersionModel.display.label(select.fromNoteVersion as any, languages) :
                select.fromProjectVersion ? ProjectVersionModel.display.label(select.fromProjectVersion as any, languages) :
                    select.fromRoutineVersion ? RoutineVersionModel.display.label(select.fromRoutineVersion as any, languages) :
                        select.fromSmartContractVersion ? SmartContractVersionModel.display.label(select.fromSmartContractVersion as any, languages) :
                            select.fromStandardVersion ? StandardVersionModel.display.label(select.fromStandardVersion as any, languages) :
                                ''
        const to = select.toApi ? ApiModel.display.label(select.toApi as any, languages) :
            select.toNote ? NoteModel.display.label(select.toNote as any, languages) :
                select.toProject ? ProjectModel.display.label(select.toProject as any, languages) :
                    select.toRoutine ? RoutineModel.display.label(select.toRoutine as any, languages) :
                        select.toSmartContract ? SmartContractModel.display.label(select.toSmartContract as any, languages) :
                            select.toStandard ? StandardModel.display.label(select.toStandard as any, languages) :
                                ''
        return `${from} -> ${to}`
    },
})

export const PullRequestModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.pull_request,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})