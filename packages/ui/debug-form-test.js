// Quick debug script to test the form fixture structure
import { teamFormTestFactory } from './src/__test/fixtures/form-testing/TeamFormTest.js';

console.log('Available formFixtures:', Object.keys(teamFormTestFactory.config.formFixtures));
console.log('Minimal fixture:', JSON.stringify(teamFormTestFactory.config.formFixtures.minimal, null, 2));

try {
    const result = await teamFormTestFactory.testFormValidation("minimal", {
        isCreate: true,
        shouldPass: true,
    });
    console.log('Validation result:', result);
} catch (error) {
    console.error('Error:', error.message);
    console.error('Stack:', error.stack);
}