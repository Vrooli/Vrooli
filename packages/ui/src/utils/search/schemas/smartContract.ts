import { SmartContractSortBy } from "@shared/consts";
import { smartContractFindMany } from "api/generated/endpoints/smartContract";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const smartContractSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchSmartContract', lng),
    containers: [
        hasCompleteVersionContainer,
        votesContainer,
        bookmarksContainer,
        languagesVersionContainer,
        tagsContainer,
    ],
    fields: [
        ...hasCompleteVersionFields,
        ...votesFields,
        ...bookmarksFields,
        ...languagesVersionFields,
        ...tagsFields,
    ]
})

export const smartContractSearchParams = (lng: string) => toParams(smartContractSearchSchema(lng), smartContractFindMany, SmartContractSortBy, SmartContractSortBy.ScoreDesc);