import { shapeTag } from "./tag";
import { createPrims, createRel } from "./tools";
export const shapeFocusModeFilter = {
    create: (d) => ({
        ...createPrims(d, "id", "filterType"),
        ...createRel(d, "focusMode", ["Connect"], "one"),
        ...createRel(d, "tag", ["Create", "Connect"], "one", shapeTag),
    }),
};
//# sourceMappingURL=focusModeFilter.js.map