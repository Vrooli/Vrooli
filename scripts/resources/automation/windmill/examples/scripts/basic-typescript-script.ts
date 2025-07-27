/**
 * Basic TypeScript Script Example for Windmill
 * 
 * This demonstrates a simple function that can be deployed as a Windmill script.
 * It shows input validation, type safety, and return value handling.
 */

// Define the input interface for better type safety
interface GreetingInput {
    name: string;
    title?: string;
    language?: 'en' | 'es' | 'fr' | 'de';
}

// Main function that Windmill will execute
export function main(input: GreetingInput): { greeting: string; timestamp: string; metadata: object } {
    // Input validation
    if (!input.name || input.name.trim() === '') {
        throw new Error('Name is required and cannot be empty');
    }
    
    // Default values
    const title = input.title || '';
    const language = input.language || 'en';
    
    // Greeting messages in different languages
    const greetings = {
        en: 'Hello',
        es: 'Hola',
        fr: 'Bonjour',
        de: 'Hallo'
    };
    
    // Build the greeting
    const greeting = `${greetings[language]} ${title} ${input.name}!`.trim();
    
    // Return structured response
    return {
        greeting,
        timestamp: new Date().toISOString(),
        metadata: {
            language,
            title: title || 'No title provided',
            characterCount: greeting.length,
            wordCount: greeting.split(' ').length
        }
    };
}

// Example usage when run manually:
// Input: { "name": "Alice", "title": "Dr.", "language": "en" }
// Output: {
//   "greeting": "Hello Dr. Alice!",
//   "timestamp": "2025-01-26T...",
//   "metadata": {
//     "language": "en",
//     "title": "Dr.",
//     "characterCount": 15,
//     "wordCount": 3
//   }
// }