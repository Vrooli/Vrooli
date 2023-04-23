import { exists } from "@local/utils";
export const fragmentsNeeded = (fragments, partialTag) => {
    const temp1 = fragments.map(([fragmentName]) => fragmentName);
    console.log("fragments needed start", temp1, temp1.length);
    const getDependentFragments = (fragmentName, fragments) => {
        const fragmentTag = fragments.find(([name]) => name === fragmentName)?.[1];
        if (!fragmentTag)
            return [];
        const nestedFragments = fragmentTag.match(/\.\.\.(\w+)/g);
        if (!exists(nestedFragments))
            return [];
        let dependentFragments = [];
        nestedFragments.forEach(fragment => {
            const name = fragment.replace("...", "");
            dependentFragments.push(name);
            dependentFragments = [...dependentFragments, ...getDependentFragments(name, fragments)];
        });
        return Array.from(new Set(dependentFragments));
    };
    const inPartial = fragments.filter(([fragmentName]) => partialTag.includes(fragmentName));
    const allFragmentsUsed = new Set(inPartial.map(([fragmentName]) => fragmentName));
    console.log("fragments needed inPartial", Array.from(allFragmentsUsed), allFragmentsUsed.size);
    inPartial.forEach(([fragmentName]) => {
        const dependentFragments = getDependentFragments(fragmentName, fragments);
        dependentFragments.forEach(fragment => {
            allFragmentsUsed.add(fragment);
        });
    });
    const temp2 = fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName)).map(([fragmentName]) => fragmentName);
    console.log("fragments needed result", temp2, temp2.length);
    return fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName));
};
//# sourceMappingURL=fragmentsNeeded.js.map