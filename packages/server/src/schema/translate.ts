import { gql } from 'apollo-server-express';
import { CustomError } from '../events/error';
import { Translate, TranslateInput } from './types';
import { IWrap } from '../types';
import { Context } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import fetch from 'node-fetch';

export const typeDef = gql`

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
`

export const resolvers = {
    Query: {
        translate: async (_parent: undefined, { input }: IWrap<TranslateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Translate> => {
            throw new CustomError('0328', 'NotImplemented', req.languages);
            // Get IETF subtags for source and target languages
            const sourceTag = input.languageSource.split('-')[0];
            const targetTag = input.languageTarget.split('-')[0];
            // Try to parse fields object from stringified JSON
            let fields: { [key: string]: string } = {};
            try {
                fields = JSON.parse(input.fields);
            } catch (e) {
                throw new CustomError('0329', 'InvalidArgs', req.languages);
            }
            // Grab translatable values from input
            const filteredFields = Object.entries(fields).filter(([key, value]) => !['__typename', 'id', 'language'].includes(key) && typeof value === 'string' && value.trim().length > 0);
            // If there are no fields, return empty object
            if (Object.keys(filteredFields).length === 0) {
                return {
                    fields: JSON.stringify({}),
                    language: targetTag,
                }
            }
            // Use LibreTranslate API to translate fields. 
            // Must make a call for each field, using promise all
            const promises = filteredFields.map(async ([key, value]) => {
                console.log('in promise', value.trim(), encodeURI(value.trim()));
                const url = `http://localhost:${process.env.PORT_TRANSLATE}/translate?source=${sourceTag}&target=${targetTag}&q=${encodeURI(value.trim())}`;
                console.log('translate url', url);
                const response = await fetch(url);
                const json = await response.json() as any;
                console.log('got libretranslate response', JSON.stringify(json));
                return {
                    [key]: json?.translatedText,
                }
            });
            const results = await Promise.all(promises);
            console.log('translate results', JSON.stringify(results));
            // Combine results into one object
            const translatedFields = results.reduce((acc, cur) => {
                return {
                    ...acc,
                    ...cur,
                }
            }, {});
            return {
                fields: JSON.stringify(translatedFields),
                language: targetTag,
            }
        },
    },
}