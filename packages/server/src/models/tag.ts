import { Tag, TagInput } from "schema/types";
import { creater, deleter, findByIder, MODEL_TYPES, reporter, updater } from "./base";

export function TagModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Tag
    }

    return {
        ...obj,
        ...findByIder<Tag>(obj),
        ...creater<TagInput, Tag>(obj),
        ...updater<TagInput, Tag>(obj),
        ...deleter(obj),
        ...reporter(obj)
    }
}