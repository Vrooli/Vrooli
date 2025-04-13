import { endpointsRoutine, FormSchema, InputType, RoutineSortBy, RoutineType } from "@local/shared";
import { Icon, IconInfo } from "../../../icons/Icons.js";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common.js";

export type RoutineTypeOption = {
    type: RoutineType;
    label: string;
    description: string;
    iconInfo: IconInfo;
};

export const routineTypes = [
    {
        type: RoutineType.Informational,
        label: "Basic",
        description: "Has no side effects. Used to collect information, provide instructions, or as a placeholder.",
        iconInfo: { name: "Help", type: "Common" } as const,
    },
    {
        type: RoutineType.MultiStep,
        label: "Multi-step",
        description: "A combination of other routines, using a graph to define the order of execution.",
        iconInfo: { name: "Routine", type: "Routine" } as const,
    },
    {
        type: RoutineType.Generate,
        label: "Generate",
        description: "Sends inputs to an AI (e.g. GPT-4o) and returns its output.",
        iconInfo: { name: "Magic", type: "Common" } as const,
    },
    {
        type: RoutineType.Data,
        label: "Data",
        description: "Contains a single output and nothing else. Useful for providing hard-coded data to other routines, such as a prompt for a \"Generate\" routine.",
        iconInfo: { name: "CaseSensitive", type: "Text" } as const,
    },
    {
        type: RoutineType.Action,
        label: "Action",
        description: "Performs specific actions within Vrooli, such as creating, updating, or deleting objects.",
        iconInfo: { name: "Action", type: "Common" } as const,
    },
    {
        type: RoutineType.Code,
        label: "Code",
        description: "Runs code to convert inputs to outputs. Useful for converting plaintext to structured data.",
        iconInfo: { name: "Terminal", type: "Common" } as const,
    },
    {
        type: RoutineType.Api,
        label: "API",
        description: "Sends inputs to an API and returns its output. Useful for connecting to external services.",
        iconInfo: { name: "Api", type: "Common" } as const,
    },
    {
        type: RoutineType.Web,
        label: "Web",
        description: "Searches the web for information.",
        iconInfo: { name: "Web", type: "Common" } as const,
    },
    //{
    //     type: RoutineType.SmartContract,
    //     label: "Smart Contract",
    //     description: "Connects to a smart contract on the blockchain, sending inputs and returning outputs.",
    //     iconInfo: { name: "SmartContract", type: "Common" } as const,
    // },
];
export function getRoutineTypeLabel(option: RoutineTypeOption) {
    return option.label;
}
export function getRoutineTypeDescription(option: RoutineTypeOption) {
    return option.description;
}
export function getRoutineTypeIcon(option: RoutineTypeOption) {
    return <Icon
        decorative
        info={option.iconInfo}
    />;
}

export function routineSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRoutine"),
        containers: [
            {
                direction: "column",
                disableCollapse: true,
                totalItems: 1,
            },
            hasCompleteVersionContainer,
            votesContainer(),
            bookmarksContainer(),
            languagesVersionContainer(),
            tagsContainer(),
        ],
        elements: [
            {
                fieldName: "latestVersionRoutineType",
                id: "latestVersionRoutineType",
                label: "Routine Type",
                type: InputType.Selector,
                props: {
                    defaultValue: null,
                    getOptionDescription: (option) => option.description,
                    getOptionLabel: (option) => option.label,
                    getOptionValue: (option) => option.label,
                    options: [
                        {
                            type: null,
                            label: "All",
                            iconInfo: { name: "Help", type: "Common" } as const,
                        },
                        ...routineTypes,
                    ],
                },
            },
            ...hasCompleteVersionFields(),
            ...votesFields(),
            ...bookmarksFields(),
            ...languagesVersionFields(),
            ...tagsFields(),
        ],
    };
}

export function routineSearchParams() {
    return toParams(routineSearchSchema(), endpointsRoutine, RoutineSortBy, RoutineSortBy.ScoreDesc);
}

