# Writing Effective Tests

This document provides guidelines and best practices for writing unit and integration tests for the Vrooli platform using Mocha, Chai, and Sinon. Adhering to these practices will help ensure our tests are reliable, maintainable, and provide good coverage.

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
-   **Assert:** Verify that the outcome of the action is as expected. Use Chai assertions to check conditions.

```typescript
import { expect } from 'chai';
import sinon from 'sinon';
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

## 4. Using Chai for Assertions

Chai provides a rich set of assertion styles. We primarily use the **Expect** and **Should** BDD styles.

-   **`expect(target).to...`**
-   **`target.should...`**

Refer to the [Chai Assertion Guide](https://www.chaijs.com/guide/styles/#expect) for a full list of assertions.

**Common Assertions:**

-   `expect(value).to.equal(expected);` (strict equality ===)
-   `expect(object).to.deep.equal(expectedObject);` (deep equality for objects/arrays)
-   `expect(value).to.be.true;` / `expect(value).to.be.false;`
-   `expect(value).to.be.null;` / `expect(value).to.be.undefined;`
-   `expect(array).to.include(member);`
-   `expect(array).to.have.lengthOf(expectedLength);`
-   `expect(fn).to.throw(ErrorType, /message pattern/);`

## 5. Using Sinon for Mocks, Spies, and Stubs

Sinon.JS helps isolate the code under test by replacing its dependencies with controllable test doubles.

-   **Spies (`sinon.spy()`):** Record information about function calls (how many times called, with what arguments, etc.) without affecting their behavior.
    ```typescript
    const myObject = { log: () => console.log('logged') };
    const logSpy = sinon.spy(myObject, 'log');
    myObject.log();
    expect(logSpy.calledOnce).to.be.true;
    logSpy.restore(); // Important to restore original method
    ```

-   **Stubs (`sinon.stub()`):** Replace functions and control their behavior (e.g., make them return specific values, throw errors, or execute custom logic). Stubs also have all the spying capabilities.
    ```typescript
    const fs = require('fs');
    const readFileStub = sinon.stub(fs, 'readFile').yields(null, 'file content'); // For async callback
    // const readFileStub = sinon.stub(fs, 'readFile').resolves('file content'); // For Promise-based async

    // Act: call function that uses fs.readFile

    expect(readFileStub.calledWith('/path/to/file')).to.be.true;
    readFileStub.restore();
    ```
    -   **Controlling Stub Behavior:**
        -   `stub.returns(value)`: Makes the stub return a specific value.
        -   `stub.throws(new Error('Oops'))`: Makes the stub throw an error.
        -   `stub.resolves(value)`: Makes the stub return a resolved promise.
        -   `stub.rejects(new Error('Failed'))`: Makes the stub return a rejected promise.
        -   `stub.yields(arg1, arg2, ...)`: Invokes a callback passed to the stub.

-   **Mocks (`sinon.mock()`):** Create test doubles with pre-programmed expectations. Mocks are stricter than stubs and will fail if the specified expectations are not met. Use sparingly, as they can lead to more brittle tests.
    ```typescript
    const myApi = { fetchData: () => { /* ... */ } };
    const mockApi = sinon.mock(myApi);
    mockApi.expects('fetchData').once().withArgs('endpoint').returns({ data: 'success' });

    // Act: Code that calls myApi.fetchData('endpoint')

    mockApi.verify(); // Verifies all expectations were met
    mockApi.restore();
    ```

-   **Sandbox (`sinon.createSandbox()`):** Useful for managing multiple spies/stubs. `sandbox.restore()` restores all test doubles created within that sandbox.
    ```typescript
    describe('MyService', () => {
        let sandbox: sinon.SinonSandbox;

        beforeEach(() => {
            sandbox = sinon.createSandbox();
        });

        afterEach(() => {
            sandbox.restore();
        });

        it('should do something with a stub', () => {
            const stubbedMethod = sandbox.stub(someObject, 'methodToStub').returns('stubbed value');
            // ... test logic ...
        });
    });
    ```

-   **Using `sinon-chai`:** Integrates Sinon with Chai for more expressive assertions like:
    -   `expect(spy).to.have.been.called;`
    -   `expect(spy).to.have.been.calledWith(arg1, arg2);`
    -   `expect(stub).to.have.returned(value);`

## 6. Test Data Management

-   Use realistic and varied test data.
-   For complex objects, consider creating helper functions or factories to generate test data.
-   Keep test data close to the tests that use it, or in clearly named fixture files if shared across many tests.
-   Avoid hardcoding large, complex data directly within test cases if it impacts readability.

## 7. Asynchronous Tests

Mocha handles asynchronous tests well.

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