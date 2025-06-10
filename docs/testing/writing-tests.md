# Writing Effective Tests

This document provides guidelines and best practices for writing unit and integration tests for the Vrooli platform using Vitest. Adhering to these practices will help ensure our tests are reliable, maintainable, and provide good coverage.

## 1. Guiding Principles

-   **Readable:** Tests should be easy to understand. Use clear descriptions and structure your tests logically.
-   **Reliable:** Tests should produce consistent results. Avoid flaky tests that pass or fail intermittently.
-   **Maintainable:** Tests should be easy to update as the codebase evolves. Avoid overly complex or brittle tests.
-   **Fast:** Tests should run quickly to provide fast feedback to developers.
-   **Focused:** Each test case should typically verify a single behavior or aspect of the code.
-   **Independent:** Tests should not depend on each other or the order of execution.

## 2. Test Structure (Arrange-Act-Assert)

A common pattern for structuring test cases is Arrange-Act-Assert (AAA):

-   **Arrange:** Set up the necessary preconditions and inputs. This might involve initializing objects, mocking dependencies, or preparing test data.
-   **Act:** Execute the code under test with the arranged parameters.
-   **Assert:** Verify that the outcome of the action is as expected. Use Vitest assertions to check conditions.

```typescript
import { describe, it, expect, vi } from 'vitest';
// import your module to test
// import { functionToTest, classToTest } from '../your-module';

describe('MyModule', () => {
    describe('functionToTest', () => {
        it('should return true when condition is met', () => {
            // Arrange
            const input = /* some input */;
            const expectedOutput = true;

            // Act
            // const actualOutput = functionToTest(input);

            // Assert
            // expect(actualOutput).to.equal(expectedOutput);
            expect(true).to.equal(true); // Placeholder
        });

        it('should call a dependency correctly', () => {
            // Arrange
            const mockDependency = {
                doSomething: sinon.stub().returns(42)
            };
            // const myInstance = new classToTest(mockDependency);

            // Act
            // myInstance.callDependencyMethod();

            // Assert
            // expect(mockDependency.doSomething.calledOnce).to.be.true;
            // expect(mockDependency.doSomething.calledWith(/* expected args */)).to.be.true;
            expect(true).to.equal(true); // Placeholder
        });
    });
});
```

## 3. Using Mocha

-   **`describe(title, callback)`:** Groups related test cases. Can be nested for further organization.
-   **`it(title, callback)`:** Defines an individual test case.
-   **Hooks:** Mocha provides hooks to set up preconditions or clean up after tests:
    -   `before()`: Runs once before all tests in a `describe` block.
    -   `after()`: Runs once after all tests in a `describe` block.
    -   `beforeEach()`: Runs before each `it` test case in a `describe` block.
    -   `afterEach()`: Runs after each `it` test case in a `describe` block.

    Use `beforeEach` and `afterEach` to ensure tests are independent by resetting state or mocks.

## 4. Using Vitest Assertions

Vitest provides a comprehensive set of assertions through its `expect` API.

**Common Assertions:**

-   `expect(value).toBe(expected);` (strict equality ===)
-   `expect(object).toEqual(expectedObject);` (deep equality for objects/arrays)
-   `expect(value).toBeTruthy();` / `expect(value).toBeFalsy();`
-   `expect(value).toBeNull();` / `expect(value).toBeUndefined();`
-   `expect(array).toContain(member);`
-   `expect(array).toHaveLength(expectedLength);`
-   `expect(fn).toThrow(ErrorType);` / `expect(fn).toThrow(/message pattern/);`

## 5. Using Vitest for Mocks, Spies, and Stubs

Vitest provides built-in mocking capabilities through its `vi` utility to isolate the code under test.

-   **Spies (`vi.spyOn()`):** Record information about function calls (how many times called, with what arguments, etc.) without affecting their behavior.
    ```typescript
    const myObject = { log: () => console.log('logged') };
    const logSpy = vi.spyOn(myObject, 'log');
    myObject.log();
    expect(logSpy).toHaveBeenCalledOnce();
    logSpy.mockRestore(); // Important to restore original method
    ```

-   **Stubs/Mocks (`vi.fn()`, `vi.spyOn()`):** Replace functions and control their behavior.
    ```typescript
    import * as fs from 'fs';
    const readFileStub = vi.spyOn(fs, 'readFile').mockResolvedValue('file content'); // For Promise-based
    // const readFileStub = vi.fn().mockImplementation((path, cb) => cb(null, 'file content')); // For callback

    // Act: call function that uses fs.readFile

    expect(readFileStub).toHaveBeenCalledWith('/path/to/file');
    readFileStub.mockRestore();
    ```
    -   **Controlling Mock Behavior:**
        -   `mock.mockReturnValue(value)`: Makes the mock return a specific value.
        -   `mock.mockImplementation(() => { throw new Error('Oops'); })`: Makes the mock throw an error.
        -   `mock.mockResolvedValue(value)`: Makes the mock return a resolved promise.
        -   `mock.mockRejectedValue(new Error('Failed'))`: Makes the mock return a rejected promise.

-   **Module Mocks (`vi.mock()`):** Mock entire modules.
    ```typescript
    vi.mock('./myApi', () => ({
        fetchData: vi.fn().mockResolvedValue({ data: 'success' })
    }));

    // Act: Code that calls imported fetchData('endpoint')

    expect(fetchData).toHaveBeenCalledWith('endpoint');
    ```

-   **Cleanup:** Vitest automatically clears all mocks between tests when `restoreMocks: true` is set in config.
    ```typescript
    describe('MyService', () => {
        afterEach(() => {
            vi.clearAllMocks(); // Clear mock history
            // or vi.restoreAllMocks(); // Restore original implementations
        });

        it('should do something with a mock', () => {
            const mockedMethod = vi.spyOn(someObject, 'methodToStub').mockReturnValue('mocked value');
            // ... test logic ...
        });
    });
    ```

-   **Common Mock Assertions:**
    -   `expect(spy).toHaveBeenCalled();`
    -   `expect(spy).toHaveBeenCalledWith(arg1, arg2);`
    -   `expect(spy).toHaveBeenCalledTimes(n);`
    -   `expect(spy).toHaveReturnedWith(value);`

## 6. Test Data Management

-   Use realistic and varied test data.
-   For complex objects, consider creating helper functions or factories to generate test data.
-   Keep test data close to the tests that use it, or in clearly named fixture files if shared across many tests.
-   Avoid hardcoding large, complex data directly within test cases if it impacts readability.

## 7. Asynchronous Tests

Vitest handles asynchronous tests seamlessly.

-   **Callbacks:** If your test function takes a `done` callback, Mocha will wait until `done()` is called.
    ```typescript
    it('should complete an async operation (callback)', (done) => {
        asyncOperation((error, result) => {
            if (error) return done(error);
            expect(result).to.equal('expected');
            done();
        });
    });
    ```

-   **Promises:** If your test function returns a Promise, Mocha will wait for the promise to resolve or reject.
    ```typescript
    it('should complete an async operation (promise)', () => {
        return asyncPromiseOperation().then(result => {
            expect(result).to.equal('expected');
        });
    });

    // Or using async/await (preferred)
    it('should complete an async operation (async/await)', async () => {
        const result = await asyncPromiseOperation();
        expect(result).to.equal('expected');
    });
    ```

## 8. What to Test (and What Not To)

-   **Do Test:**
    -   Public API of your modules/classes.
    -   Business logic and algorithms.
    -   Boundary conditions and edge cases.
    -   Error handling paths.
    -   Interactions with dependencies (use mocks/stubs).
-   **Don't Test (Generally):**
    -   Private methods directly (test them via public methods).
    -   Third-party libraries (assume they are tested; test your integration with them, not their internal logic).
    -   Trivial code (e.g., simple getters/setters that just return a value without logic), unless they contain critical logic.
    -   Overly complex scenarios in a single unit test; break them down.

## 9. Test File Organization

-   Test files should typically reside alongside the code they are testing (e.g., `myModule.ts` and `myModule.test.ts` in the same directory) or in a dedicated `__tests__` or `test` subdirectory.
-   Current project convention appears to be `*.test.ts` in the same directory or related subdirectories within `src`.

By following these guidelines, we can build a robust and effective test suite that supports the development and maintenance of Vrooli. 