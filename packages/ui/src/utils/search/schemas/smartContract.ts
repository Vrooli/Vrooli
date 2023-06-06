import { endpointGetSmartContracts, SmartContractSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const smartContractSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchSmartContract"),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ],
});

export const smartContractSearchParams = () => toParams(smartContractSearchSchema(), endpointGetSmartContracts, SmartContractSortBy, SmartContractSortBy.ScoreDesc);
