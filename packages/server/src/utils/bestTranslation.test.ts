/* eslint-disable @typescript-eslint/ban-ts-comment */
import { bestTranslation } from "./bestTranslation";

describe('bestTranslation', () => {
    test('returns correct translation for preferred language match', () => {
        const translations = [{ language: 'en', text: 'Hello' }, { language: 'fr', text: 'Bonjour' }];
        const languages = ['fr', 'en'];
        expect(bestTranslation(translations, languages)).toEqual({ language: 'fr', text: 'Bonjour' });
    });

    test('returns first translation if no preferred languages match', () => {
        const translations = [{ language: 'en', text: 'Hello' }, { language: 'fr', text: 'Bonjour' }];
        const languages = ['es', 'de'];
        expect(bestTranslation(translations, languages)).toEqual({ language: 'en', text: 'Hello' });
    });

    test('returns undefined for empty translations array', () => {
        const translations = [];
        const languages = ['en', 'fr'];
        expect(bestTranslation(translations, languages)).toBeUndefined();
    });

    test('returns first translation if no preferred languages are provided', () => {
        const translations = [{ language: 'en', text: 'Hello' }, { language: 'fr', text: 'Bonjour' }];
        const languages = [];
        expect(bestTranslation(translations, languages)).toEqual({ language: 'en', text: 'Hello' });
    });

    test('respects the order of preferred languages', () => {
        const translations = [{ language: 'es', text: 'Hola' }, { language: 'en', text: 'Hello' }];
        const languages = ['en', 'es'];
        expect(bestTranslation(translations, languages)).toEqual({ language: 'en', text: 'Hello' });
    });

    test('returns the first translation when multiple matches exist', () => {
        const translations = [{ language: 'en', text: 'Hello' }, { language: 'en', text: 'Hi' }];
        const languages = ['en'];
        expect(bestTranslation(translations, languages)).toEqual({ language: 'en', text: 'Hello' });
    });

    test('handles undefined or null inputs gracefully', () => {
        // @ts-ignore: Testing runtime scenario
        expect(bestTranslation(undefined, ['en'])).toBeUndefined();
        // @ts-ignore: Testing runtime scenario
        expect(bestTranslation(null, ['en'])).toBeUndefined();
        // @ts-ignore: Testing runtime scenario
        expect(bestTranslation([{ language: 'en', text: 'Hello' }], undefined)).toEqual({ language: 'en', text: 'Hello' });
        // @ts-ignore: Testing runtime scenario
        expect(bestTranslation([{ language: 'en', text: 'Hello' }], null)).toEqual({ language: 'en', text: 'Hello' });
    });

    test('is case-sensitive for language codes', () => {
        const translations = [{ language: 'EN', text: 'Hello' }, { language: 'fr', text: 'Bonjour' }];
        const languages = ['en', 'fr'];
        expect(bestTranslation(translations, languages)).toEqual({ language: 'fr', text: 'Bonjour' });
    });
});
