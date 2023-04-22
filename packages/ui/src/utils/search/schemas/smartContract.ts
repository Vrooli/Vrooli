import { SmartContractSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { smartContractFindMany } from "../../../api/generated/endpoints/smartContract_findMany";
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
    ]
})

export const smartContractSearchParams = () => toParams(smartContractSearchSchema(), smartContractFindMany, SmartContractSortBy, SmartContractSortBy.ScoreDesc);