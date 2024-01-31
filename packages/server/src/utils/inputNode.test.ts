import { InputNode } from "./inputNode";

describe("InputNode", () => {
    const typename = "RoutineVersion";
    const id = "testId";
    const action = "Create";

    // Test constructor behavior
    it("should correctly initialize properties with constructor", () => {
        const node = new InputNode(typename, id, action);
        expect(node.__typename).toBe(typename);
        expect(node.id).toBe(id);
        expect(node.action).toBe(action);
        expect(node.children).toHaveLength(0);
        expect(node.parent).toBeNull();
    });

    // Test adding children
    it("should correctly add children and set parent", () => {
        const parentNode = new InputNode(typename, id, action);
        const childNode = new InputNode(typename, "childId", action);

        parentNode.children.push(childNode);
        childNode.parent = parentNode;

        expect(parentNode.children).toHaveLength(1);
        expect(parentNode.children[0]).toBe(childNode);
        expect(childNode.parent).toBe(parentNode);
    });

    // Test setting parent
    it("should correctly set the parent of a node", () => {
        const parentNode = new InputNode(typename, "parentId", action);
        const childNode = new InputNode(typename, "childId", action);

        childNode.parent = parentNode;

        expect(childNode.parent).toBe(parentNode);
    });
});
