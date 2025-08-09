// Jest test setup for Agent Metareasoning Manager API

// Mock console methods to reduce test noise
const originalLog = console.log;
const originalError = console.error;

beforeAll(() => {
  // Suppress console.log during tests unless in verbose mode
  if (!process.env.JEST_VERBOSE) {
    console.log = jest.fn();
  }
  
  // Keep error logging but suppress connection errors during testing
  console.error = jest.fn((message, ...args) => {
    if (typeof message === 'string' && message.includes('ECONNREFUSED')) {
      // Suppress connection refused errors during testing
      return;
    }
    originalError.call(console, message, ...args);
  });
});

afterAll(() => {
  // Restore console methods
  console.log = originalLog;
  console.error = originalError;
});

// Setup test environment variables
process.env.NODE_ENV = 'test';
process.env.METAREASONING_API_PORT = '8093';
process.env.POSTGRES_HOST = 'localhost';
process.env.POSTGRES_PORT = '5432';
process.env.POSTGRES_DB = 'postgres';
process.env.POSTGRES_USER = 'postgres';
process.env.REDIS_PORT = '6379';
process.env.N8N_PORT = '5678';
process.env.WINDMILL_PORT = '8000';