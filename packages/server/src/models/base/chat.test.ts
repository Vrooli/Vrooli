
import { chatCreateNestedMessages } from "./chat";

describe('chatCreateNestedMessages', () => {
    test('should create a single nested create with no parent when no lastSequenceId and no parent.connect.id', () => {
        const messages = [
            { id: '1', content: 'Hello', randomField: 'random' },
            { id: '2', content: 'How are you?' }
        ];
        const branchInfo = { isBranchable: false, lastSequenceId: null };
        // Starts at the end of the array. No parents exist, so it should just create a single nested create.
        // Since the last message (i.e. the first in the array) has no parent, and lastSequenceId is null, it should not have a final parent create
        const expected = { create: { id: '2', content: 'How are you?', parent: { create: { id: '1', content: 'Hello', randomField: 'random' } } } };
        const result = chatCreateNestedMessages(messages, branchInfo);
        expect(result).toEqual(expected);
    });

    test('should use lastSequenceId for the last parent in a non-branchable chat', () => {
        const messages = [
            { id: '1', content: 'Hello' },
            { id: '2', content: 'How are you?' }
        ];
        const branchInfo = { isBranchable: false, lastSequenceId: '123' };
        const result = chatCreateNestedMessages(messages, branchInfo);
        // Starts at the end of the array. No parents exist, so it should just create a single nested create. 
        // Since the last message (i.e. the first in the array) has no parent, it should connect to the lastSequenceId
        const expected = { create: { id: '2', content: 'How are you?', parent: { create: { id: '1', content: 'Hello', parent: { connect: { id: '123' } } } } } };
        expect(result).toEqual(expected);
    });

    test('should respect existing parent.connect.id for the first message when branching', () => {
        const messages = [
            { id: '1', content: 'First message', parent: { connect: { id: 'abc' } } },
            { id: '2', content: 'Second message' }
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        // Starts at the end of the array. No parents exist, before the last message, so it should just create a single nested create. 
        // Since the last message has a connect parent, it should use that parent instead of the lastSequenceId
        const result = chatCreateNestedMessages(messages, branchInfo);
        const expected = { create: { id: '2', content: 'Second message', parent: { create: { id: '1', content: 'First message', parent: { connect: { id: 'abc' } } } } } };
        expect(result).toEqual(expected);
    });

    test('should handle branchable chat with no specified parents correctly', () => {
        const messages = [
            { id: '1', content: 'Hello' },
            { id: '2', content: 'How are you?' }
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        // Starts at the end of the array. No parents exist, so it should just create a single nested create.
        // Since the last message (i.e. the first in the array) has no parent, it should connect to the lastSequenceId
        const result = chatCreateNestedMessages(messages, branchInfo);
        const expected = { create: { id: '2', content: 'How are you?', parent: { create: { id: '1', content: 'Hello', parent: { connect: { id: '123' } } } } } };
        expect(result).toEqual(expected);
    });

    test('should handle no messages correctly', () => {
        const messages = [];
        const branchInfo = { isBranchable: false, lastSequenceId: null };
        const result = chatCreateNestedMessages(messages, branchInfo);
        const expected = undefined;
        expect(result).toEqual(expected);
    });

    test('should handle multiple parent connects correctly - test 1', () => {
        const messages = [
            { id: '1', content: 'First message', parent: { connect: { id: 'abc' } } },
            { id: '2', content: 'Second message', parent: { connect: { id: 'def' } } },
            { id: '3', content: 'Third message' }
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        // Starts at 3. Has no parent, so connects to 2. 
        // 2 has a parent, but it doesn't exist in the array. This means it should connect to that parent, since it must already exist in the database. 
        // Now we start a new branch at 1, since 2 connected to a parent that wasn't in the array. 1 has a parent not in the array, so it should connect to that parent.
        const result = chatCreateNestedMessages(messages, branchInfo);
        const expected = {
            create: [
                { id: '3', content: 'Third message', parent: { create: { id: '2', content: 'Second message', parent: { connect: { id: 'def' } } } } },
                { id: '1', content: 'First message', parent: { connect: { id: 'abc' } } }
            ]
        };
        expect(result).toEqual(expected);
    });

    test('should handle multiple parent connects correctly - test 2', () => {
        const messages = [
            { id: '1', content: 'First message', parent: { connect: { id: 'abc' } } },
            { id: '2', content: 'Second message' },
            { id: '3', content: 'Third message', parent: { connect: { id: 'def' } } }
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        // Starts at 3. Has a parent, but it doesn't exist in the array. This means it should connect to that parent, since it must already exist in the database.
        // Now we start a new branch at 2, since 3 connected to a parent that wasn't in the array. 2 has no parent, so it should connect to 1. 
        // 1 has a parent not in the array, so it should connect to that parent.
        const result = chatCreateNestedMessages(messages, branchInfo);
        const expected = {
            create: [
                { id: '3', content: 'Third message', parent: { connect: { id: 'def' } } },
                { id: '2', content: 'Second message', parent: { create: { id: '1', content: 'First message', parent: { connect: { id: 'abc' } } } } }
            ]
        };
        expect(result).toEqual(expected);
    });

    test('should handle all parent connects correctly', () => {
        const messages = [
            { id: '1', content: 'First message', parent: { connect: { id: '3' } } },
            { id: '2', content: 'Second message', parent: { connect: { id: '1' } } },
            { id: '3', content: 'Third message', parent: { connect: { id: '4' } } },
            { id: '4', content: 'Fourth message' },
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        // Starts at 4. Has no parent, so connects to 3.
        // 3 has a parent, but it's a message we already checked. That means we'll have to throw away the current branch and start a new one at 3, and connect it to 4.
        // 4 has no parent so we connect it to 2. Note how this time we don't connect it to the message directly before it, but rather the first message before it that 
        // hasn't already been added to a create object.
        // 2 connects to 1.
        // 1 connects to 3, but this has already been checked. This means we'll have to throw away the current branch and start a new one at 1, and connect it to 3. 
        // 3 connects to 4, and 4 connects to 2. 2 connects to 1, but this has already been checked. This means we'll have to throw away the current branch and start a new one at 2, and connect it to 1.
        // 1 connects to 3, 3 connects to 4, and 4 is the final message being checked. This means we connect it to the lastSequenceId
        const result = chatCreateNestedMessages(messages, branchInfo);
        const expected = {
            create: {
                id: '2',
                content: 'Second message',
                parent: {
                    create: {
                        id: '1',
                        content: 'First message',
                        parent: {
                            create: {
                                id: '3',
                                content: 'Third message',
                                parent: {
                                    create: {
                                        id: '4',
                                        content: 'Fourth message',
                                        parent: {
                                            connect: { id: '123' }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        };
        expect(result).toEqual(expected);
    });

    test('should throw error when loop is detected', () => {
        const messages = [
            { id: '1', content: 'First message', parent: { connect: { id: '3' } } },
            { id: '2', content: 'Second message', parent: { connect: { id: '1' } } },
            { id: '3', content: 'Third message', parent: { connect: { id: '2' } } },
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        expect(() => chatCreateNestedMessages(messages, branchInfo)).toThrow();
    });

    test('should throw error when multiple messages have the same parent. This is technically valid, but we don\'t bother to support it.', () => {
        const messages = [
            { id: '1', content: 'First message', parent: { connect: { id: '3' } } },
            { id: '2', content: 'Second message', parent: { connect: { id: '3' } } },
            { id: '3', content: 'Third message' },
        ];
        const branchInfo = { isBranchable: true, lastSequenceId: '123' };
        expect(() => chatCreateNestedMessages(messages, branchInfo)).toThrow();
    });
});
