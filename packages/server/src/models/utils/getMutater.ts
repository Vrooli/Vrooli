import { CustomError } from "../../events";
import { ObjectMap } from "../builder";
import { GraphQLModelType, Mutater, MutaterShapes } from "../types";

export function getMutater<
    GQLObject extends { [x: string]: any },
    Create extends MutaterShapes | false,
    Update extends MutaterShapes | false,
    RelationshipCreate extends MutaterShapes | false,
    RelationshipUpdate extends MutaterShapes | false,
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Mutater<GQLObject, Create, Update, RelationshipCreate, RelationshipUpdate> {
    const mutater: Mutater<GQLObject, Create, Update, RelationshipCreate, RelationshipUpdate> | undefined = ObjectMap[objectType]?.mutate;
    if (!mutater) {
        throw new CustomError('0338', 'InternalError', languages, { errorTrace, objectType });
    }
    return mutater;
}