import { endpointGetRoutine, endpointGetRoutines, InputType, RoutineSortBy, RoutineType } from "@local/shared";
import { FormSchema } from "forms/types";
import { ActionIcon, ApiIcon, CaseSensitiveIcon, HelpIcon, MagicIcon, RoutineIcon, SmartContractIcon, TerminalIcon } from "icons";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const routineTypes = [
    {
        type: RoutineType.Informational,
        label: "Informational/Placeholder",
        description: "Contains no additional data. Used to provide instructions, a way to prompt users for information, or as a placeholder.",
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
        description: "Runs sandboxed JavaScript code to convert inputs to outputs. Useful for converting plaintext to structured data. Does not have access to the internet.",
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

export const routineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRoutine"),
    containers: [
        { disableCollapse: true, title: "Routine Type", totalItems: 1 },
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        {
            fieldName: "latestVersionRoutineType",
            label: "Routine Type",
            type: InputType.Selector,
            props: {
                defaultValue: null,
                getOptionDescription: (option) => option.description,
                getOptionLabel: (option) => option.label,
                name: "routineType",
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
});

export const routineSearchParams = () => toParams(routineSearchSchema(), endpointGetRoutines, endpointGetRoutine, RoutineSortBy, RoutineSortBy.ScoreDesc);
