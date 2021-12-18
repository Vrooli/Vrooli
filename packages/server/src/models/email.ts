import { Email, EmailInput } from "schema/types";
import { creater, deleter, findByIder, MODEL_TYPES, updater } from "./base";

export function EmailModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Email
    }
    
    return {
        ...obj,
        ...findByIder<Email>(obj),
        ...creater<EmailInput, Email>(obj),
        ...updater<EmailInput, Email>(obj),
        ...deleter(obj)
    }
}