// This file acts as an example of how to implement a compositional model component.
// Typically an auto-generated GraphQL type would be imported like below:
// import { Role } from "../schema/types";

import { PrismaType } from "types";
import { creater, deleter, FormatConverter, MODEL_TYPES, updater } from "./base";

// But for this example we will manually define the types
type Maybe<T> = T | null;
type Scalars = {
    ID: string;
    String: string;
    Boolean: boolean;
    Int: number;
    Float: number;
    /** Custom description for the date scalar */
    Date: any;
    /** The `Upload` scalar type represents a file upload. */
    Upload: any;
};
type Child = {
    __typename?: 'User';
    id: Scalars['ID'];
    name: Scalars['String'];
    description: Scalars['String'];
};
type ExampleInput = {
    id: Scalars['ID'];
    text: Scalars['String'];
};
type Example = {
    __typename?: 'Example';
    description?: Maybe<Scalars['String']>;
    id: Scalars['ID'];
    title: Scalars['String'];
    children1: Array<Child>;
    children2: Array<Child>;
};

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// First, define any extra TypeScript types that are needed for the model.
// GraphQL input and output types are automatically generated from the schema.
// This leaves us with the following types:
// 1. RelationshipList - List of queryable + non-queryable relationship fields
// 2. QueryablePrimitives - All queryable non-relationship fields
// 3. AllPrimitives - All queryable + non-queryable non-relationship fields
// 4. DB - The model as it appears in the database (primitive + relationship)
//    - This one's tricky because the relationships must be defined as they appear in the database,
//      not how they are represented in the GraphQL schema.

// Type 1. RelationshipList
export type ExampleRelationshipList = 'children1' | 'children2';
// Type 2. QueryablePrimitives
export type ExampleQueryablePrimitives = Omit<Example, ExampleRelationshipList>;
// Type 3. AllPrimitives
// For this example, we will say there's one primitive field that is not queryable: 'secretField'
export type ExampleAllPrimitives = ExampleQueryablePrimitives & { secretField: string };
// type 4. Database shape
// For this example, let's say that children1 is defined by a join table,
// while children2 is linked directly to the parent.
// This means that children1 must change the shape of its relationship field,
// while children2 can be picked from the auto-generated GraphQL type
export type ExampleDB = ExampleAllPrimitives & 
Pick<Example, 'children2'> &
{ 
    children1: { child: Array<Child> },
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

// Second, define the converter component.
// This converts between the model as it is stored in the database, and how it is represented in the GraphQL schema.
// Typically this is a flattening of join tables, but it could be anything.
const formatter = (): FormatConverter<Example, any> => ({
    toDB: (obj: any): any => ({ ...obj }),
    toGraphQL: (obj: any): any => ({ ...obj })
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

// Last, we define the model component.
export function ExampleModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Comment; // This would be MODEL_TYPES.Example, if it was real
    const format = formatter();

    // the return contains every composition function that applies to this model
    return {
        prisma,
        model,
        ...format,
        ...creater<ExampleInput, Example, ExampleDB>(model, format.toDB, prisma),
        ...updater<ExampleInput, Example, ExampleDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================