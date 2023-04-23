import { gql } from "apollo-server-express";
import fetch from "node-fetch";
import { CustomError } from "../events/error";
export const typeDef = gql `
    input TranslateInput {
        fields: String!
        languageSource: String!
        languageTarget: String!
    }

    type Translate {
        fields: String!
        language: String!
    }

    extend type Query {
        translate(input: TranslateInput!): Translate!
    }
`;
export const resolvers = {
    Query: {
        translate: async (_, { input }, { prisma, req }, info) => {
            throw new CustomError("0328", "NotImplemented", req.languages);
            const sourceTag = input.languageSource.split("-")[0];
            const targetTag = input.languageTarget.split("-")[0];
            let fields = {};
            try {
                fields = JSON.parse(input.fields);
            }
            catch (e) {
                throw new CustomError("0329", "InvalidArgs", req.languages);
            }
            const filteredFields = Object.entries(fields).filter(([key, value]) => !["type", "id", "language"].includes(key) && typeof value === "string" && value.trim().length > 0);
            if (Object.keys(filteredFields).length === 0) {
                return {
                    __typename: "Translate",
                    fields: JSON.stringify({}),
                    language: targetTag,
                };
            }
            const promises = filteredFields.map(async ([key, value]) => {
                console.log("in promise", value.trim(), encodeURI(value.trim()));
                const url = `http://localhost:${process.env.PORT_TRANSLATE}/translate?source=${sourceTag}&target=${targetTag}&q=${encodeURI(value.trim())}`;
                console.log("translate url", url);
                const response = await fetch(url);
                const json = await response.json();
                console.log("got libretranslate response", JSON.stringify(json));
                return {
                    [key]: json?.translatedText,
                };
            });
            const results = await Promise.all(promises);
            console.log("translate results", JSON.stringify(results));
            const translatedFields = results.reduce((acc, cur) => {
                return {
                    ...acc,
                    ...cur,
                };
            }, {});
            return {
                __typename: "Translate",
                fields: JSON.stringify(translatedFields),
                language: targetTag,
            };
        },
    },
};
//# sourceMappingURL=translate.js.map