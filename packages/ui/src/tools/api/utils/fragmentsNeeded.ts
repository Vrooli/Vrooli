import { exists } from "@shared/utils";

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
    const temp1 = fragments.map(([fragmentName]) => fragmentName);
    console.log('fragments needed start', temp1, temp1.length);

    const getDependentFragments = (fragmentName: string, fragments: [string, string][]) => {
        const fragmentTag = fragments.find(([name]) => name === fragmentName)?.[1];
        if (!fragmentTag) return [];

        const nestedFragments = fragmentTag.match(/\.\.\.(\w+)/g);
        if (!exists(nestedFragments)) return [];

        let dependentFragments: string[] = [];
        nestedFragments.forEach(fragment => {
            const name = fragment.replace('...', '');
            dependentFragments.push(name);
            dependentFragments = [...dependentFragments, ...getDependentFragments(name, fragments)];
        });

        return Array.from(new Set(dependentFragments));
    };

    const inPartial = fragments.filter(([fragmentName]) => partialTag.includes(fragmentName));

    let allFragmentsUsed = new Set(inPartial.map(([fragmentName]) => fragmentName));
    console.log('fragments needed inPartial', Array.from(allFragmentsUsed), allFragmentsUsed.size);

    inPartial.forEach(([fragmentName]) => {
        const dependentFragments = getDependentFragments(fragmentName, fragments);
        dependentFragments.forEach(fragment => {
            allFragmentsUsed.add(fragment);
        });
    });

    const temp2 = fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName)).map(([fragmentName]) => fragmentName)
    console.log('fragments needed result', temp2, temp2.length);
    return fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName));
};