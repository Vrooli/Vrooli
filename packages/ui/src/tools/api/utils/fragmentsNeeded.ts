import { exists } from "@shared/utils"

/**
 * Determines which fragments are needed for a given partial. 
 * Useful for filtering out fragments from omitted fields, since 
 * we have no better way to do this yet.
 * @param fragments list of [name, tag] for all fragments
 * @param partialTag the partial to check
 * @returns fragments list with omitted fragments removed
 */
export const fragmentsNeeded = (
    fragments: [string, string][],
    partialTag: string
) => {
    // First, find all fragments that are used in the partial. These must be included.
    const inPartial = fragments.filter(([fragmentName]) => partialTag.includes(fragmentName))
    // Find every fragment referenced by a fragment used in the partial. 
    // This is accomplished with regex, since we know that every fragment is preceded by an ellipsis. 
    // NOTE: Unions are also preceded by an ellipsis, but there is a space in between the fragment name and the ellipsis (so not a problem)
    const allFragments: string[] = []
    // Loop through each fragment used in the partial
    inPartial.forEach(([fragmentName, fragmentTag]) => {
        // Any word followed by an ellipsis is a fragment
        const nestedFragments = fragmentTag.match(/\.\.\.(\w+)/g)
        if (!exists(nestedFragments)) return;
        // Add each unique fragment to the list
        nestedFragments.forEach(fragment => {
            const fragmentName = fragment.replace('...', '')
            if (!allFragments.includes(fragmentName)) allFragments.push(fragmentName)
        })
    })
    // Combine inPartial and allFragments into a set
    const allFragmentsUsed = new Set([...inPartial.map(([fragmentName]) => fragmentName), ...allFragments])
    // Return fragment tuples that are in the set
    return fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName))
}