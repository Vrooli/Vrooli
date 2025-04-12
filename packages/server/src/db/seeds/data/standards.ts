import { SEEDED_TAGS } from "@local/shared";
import { StandardImportData } from "../../../builders/importExport.js";

const VERSION = "1.0.0" as const;

const baseRoot = {
    __version: VERSION,
    __typename: "Standard" as const,
};

const baseRootShape = {
    isPrivate: false,
    permissions: JSON.stringify({}),
    tags: [{ __typename: "Tag" as const, tag: SEEDED_TAGS.Vrooli }],
};

const baseVersion = {
    __version: VERSION,
    __typename: "StandardVersion" as const,
};

const baseVersionShape = {
    isComplete: true,
    isPrivate: false,
    versionIndex: 0,
    versionLabel: "1.0.0",
};

export const data: StandardImportData[] = [
];
