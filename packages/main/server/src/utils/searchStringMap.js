const base = ({ insensitive }) => ({ ...insensitive });
const transField = ({ insensitive, languages }, field) => ({ some: { language: languages ? { in: languages } : undefined, [field]: { ...insensitive } } });
const wrap = (field, wrapped) => ({ [field]: wrapped });
export const SearchStringMap = {
    description: (p) => base(p),
    descriptionWrapped: (p) => wrap("description", SearchStringMap.description(p)),
    details: (p) => base(p),
    detailsWrapped: (p) => wrap("details", SearchStringMap.details(p)),
    handle: (p) => base(p),
    handleWrapped: (p) => wrap("handle", SearchStringMap.handle(p)),
    label: (p) => base(p),
    labelWrapped: (p) => wrap("label", SearchStringMap.label(p)),
    labels: ({ insensitive }) => ({ some: { label: { label: { ...insensitive } } } }),
    labelsWrapped: (p) => wrap("labels", SearchStringMap.labels(p)),
    labelsOwnerWrapped: ({ insensitive }) => wrap("labels", { some: { label: { ...insensitive } } }),
    link: (p) => base(p),
    linkWrapped: (p) => wrap("link", SearchStringMap.link(p)),
    message: (p) => base(p),
    messageWrapped: (p) => wrap("message", SearchStringMap.message(p)),
    name: (p) => base(p),
    nameWrapped: (p) => wrap("name", SearchStringMap.name(p)),
    reason: (p) => base(p),
    reasonWrapped: (p) => wrap("reason", SearchStringMap.reason(p)),
    title: (p) => base(p),
    titleWrapped: (p) => wrap("title", SearchStringMap.title(p)),
    transBio: (p) => transField(p, "bio"),
    transBioWrapped: (p) => wrap("translations", transField(p, "bio")),
    transDescription: (p) => transField(p, "description"),
    transDescriptionWrapped: (p) => wrap("translations", transField(p, "description")),
    transName: (p) => transField(p, "name"),
    transNameWrapped: (p) => wrap("translations", transField(p, "name")),
    transQuestionText: (p) => transField(p, "questionText"),
    transQuestionTextWrapped: (p) => wrap("translations", transField(p, "questionText")),
    transSummary: (p) => transField(p, "summary"),
    transSummaryWrapped: (p) => wrap("translations", transField(p, "summary")),
    transResponseWrapped: (p) => wrap("translations", transField(p, "response")),
    transText: (p) => transField(p, "text"),
    transTextWrapped: (p) => wrap("translations", transField(p, "text")),
    tag: (p) => base(p),
    tagWrapped: (p) => wrap("tag", SearchStringMap.tag(p)),
    tags: ({ insensitive }) => ({ some: { tag: { tag: { ...insensitive } } } }),
    tagsWrapped: (p) => wrap("tags", SearchStringMap.tags(p)),
};
//# sourceMappingURL=searchStringMap.js.map