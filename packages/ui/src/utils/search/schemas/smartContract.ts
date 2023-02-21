import { SmartContractSortBy } from "@shared/consts";
import { smartContractFindMany } from "api/generated/endpoints/smartContract";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const smartContractSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchSmartContract', lng),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(lng),
        bookmarksContainer(lng),
        languagesVersionContainer(lng),
        tagsContainer(lng),
    ],
    fields: [
        ...hasCompleteVersionFields(lng),
        ...votesFields(lng),
        ...bookmarksFields(lng),
        ...languagesVersionFields(lng),
        ...tagsFields(lng),
    ]
})

export const smartContractSearchParams = (lng: string) => toParams(smartContractSearchSchema(lng), smartContractFindMany, SmartContractSortBy, SmartContractSortBy.ScoreDesc);