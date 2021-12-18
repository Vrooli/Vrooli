import { Comment, CommentInput } from "schema/types";
import { creater, deleter, findByIder, MODEL_TYPES, reporter, updater } from "./base";

export function CommentModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Comment
    }

    return {
        ...obj,
        ...findByIder<Comment>(obj),
        ...creater<CommentInput, Comment>(obj),
        ...updater<CommentInput, Comment>(obj),
        ...deleter(obj),
        ...reporter(obj)
    }
}