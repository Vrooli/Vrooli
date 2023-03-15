export const SmartContractVersion_nav = `fragment SmartContractVersion_nav on SmartContractVersion {
id
isLatest
isPrivate
versionIndex
versionLabel
root {
    id
    isPrivate
}
translations {
    id
    language
    description
    jsonVariable
    name
}
}`;