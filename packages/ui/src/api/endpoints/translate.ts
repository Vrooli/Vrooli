import { toQuery } from "api/utils";

export const translateEndpoint = {
    translate: toQuery('translate', 'FindByIdInput', {} as any, 'full'),//translatePartial, 'full'),
}