import { endpointsRoutine, FormSchema, InputType, RoutineSortBy, RoutineType } from "@local/shared";
import { ActionIcon, ApiIcon, CaseSensitiveIcon, HelpIcon, MagicIcon, RoutineIcon, SmartContractIcon, TerminalIcon } from "icons/common.js";
import { SvgProps } from "types";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common.js";

export type RoutineTypeOption = {
    type: RoutineType;
    label: string;
    description: string;
    Icon: (props: SvgProps) => JSX.Element;
};

export const routineTypes = [
    {
        type: RoutineType.Informational,
        label: "Basic",
        description: "Has no side effects. Used to collect information, provide instructions, or as a placeholder.",
        Icon: HelpIcon,
    }, {
        type: RoutineType.MultiStep,
        label: "Multi-step",
        description: "A combination of other routines, using a graph to define the order of execution.",
        Icon: RoutineIcon,
    }, {
        type: RoutineType.Generate,
        label: "Generate",
        description: "Sends inputs to an AI (e.g. GPT-4o) and returns its output.",
        Icon: MagicIcon,
    }, {
        type: RoutineType.Data,
        label: "Data",
        description: "Contains a single output and nothing else. Useful for providing hard-coded data to other routines, such as a prompt for a \"Generate\" routine.",
        Icon: CaseSensitiveIcon,
    }, {
        type: RoutineType.Action,
        label: "Action",
        description: "Performs specific actions within Vrooli, such as creating, updating, or deleting objects.",
        Icon: ActionIcon,
    }, {
        type: RoutineType.Code,
        label: "Code",
        description: "Runs code to convert inputs to outputs. Useful for converting plaintext to structured data.",
        Icon: TerminalIcon,
    }, {
        type: RoutineType.Api,
        label: "API",
        description: "Sends inputs to an API and returns its output. Useful for connecting to external services.",
        Icon: ApiIcon,
    }, {
        type: RoutineType.SmartContract,
        label: "Smart Contract",
        description: "Connects to a smart contract on the blockchain, sending inputs and returning outputs.",
        Icon: SmartContractIcon,
    },
];
export function getRoutineTypeLabel(option: RoutineTypeOption) { return option.label; }
export function getRoutineTypeDescription(option: RoutineTypeOption) { return option.description; }
export function getRoutineTypeIcon(option: RoutineTypeOption) { return option.Icon; }

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
                            Icon: HelpIcon,
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

