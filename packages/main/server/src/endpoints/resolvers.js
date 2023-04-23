import { CustomError } from "../events";
export const resolveUnion = (object) => {
    if (object.__typename) {
        return object.__typename;
    }
    else {
        throw new CustomError("0364", "InternalError", ["en"]);
    }
};
//# sourceMappingURL=resolvers.js.map