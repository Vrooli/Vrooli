/* eslint-disable @typescript-eslint/ban-ts-comment */
import fs from "fs";
import path from "path";
import { DEFAULT_LANGUAGE, LLM_CONFIG_LOCATION, getActionConfig, importConfig, llmActions } from "./config";

describe('importConfig', () => {
    const configFiles = fs.readdirSync(path.join(__dirname, LLM_CONFIG_LOCATION)).filter(file => file.endsWith('.ts'));

    configFiles.forEach(async file => {
        test(`ensures all actions are present in ${file}`, async () => {
            const config = await importConfig(file);
            llmActions.forEach(action => {
                expect(config[action]).toBeDefined();
                expect(typeof config[action]).toBe('function');
            });
        });
    });

    test(`falls back to ${DEFAULT_LANGUAGE} config when a language config is not found`, async () => {
        const config = await importConfig('nonexistent-language');
        expect(config).toBeDefined();
        expect(config).toEqual(await importConfig(DEFAULT_LANGUAGE));
    });
});

describe('getActionConfig', () => {
    const botSettings1 = {
        model: "gpt-3.5-turbo",
        maxTokens: 100,
        name: "Valyxa",
        translations: {
            en: {
                bias: "",
                creativity: 0.5,
                domainKnowledge: "Planning, scheduling, and task management",
                keyPhrases: "",
                occupation: "Vrooli assistant",
                persona: "Helpful, friendly, and professional",
                startingMessage: "Hello! How can I help you today?",
                tone: "Friendly",
                verbosity: 0.5,
            },
        },
    };

    test('works for an existing language', async () => {
        const actionConfig = await getActionConfig('Start', botSettings1, DEFAULT_LANGUAGE);
        expect(actionConfig).toBeDefined();
        // Perform additional checks on the structure of actionConfig if necessary
    });

    test('works for a language that\'s not in the config', async () => {
        const actionConfig = await getActionConfig('RoutineCreate', botSettings1, 'nonexistent-language');
        expect(actionConfig).toBeDefined();
        // Since it falls back to English, compare it with the English config for the same action
        const englishConfig = await getActionConfig('RoutineCreate', botSettings1, DEFAULT_LANGUAGE);
        expect(actionConfig).toEqual(englishConfig);
    });

    test('returns an empty object for an invalid action or action that doesn\'t appear in the config', async () => {
        // @ts-ignore: Testing runtime scenario
        const actionConfig = await getActionConfig('InvalidAction', 'en', botSettings1);
        expect(actionConfig).toEqual({});
    });
});
