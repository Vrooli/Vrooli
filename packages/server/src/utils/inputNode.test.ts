import { expect } from "chai";
import { InputNode } from "./inputNode.js";

describe("InputNode", () => {
    const typename = "RoutineVersion";
    const id = "testId";
    const action = "Create";

    // Test constructor behavior
    it("should correctly initialize properties with constructor", () => {
        const node = new InputNode(typename, id, action);
        expect(node.__typename).to.equal(typename);
        expect(node.id).to.equal(id);
        expect(node.action).to.equal(action);
        expect(node.children).to.have.lengthOf(0);
        expect(node.parent).to.be.null;
    });

    // Test adding children
    it("should correctly add children and set parent", () => {
        const parentNode = new InputNode(typename, id, action);
        const childNode = new InputNode(typename, "childId", action);

        parentNode.children.push(childNode);
        childNode.parent = parentNode;

        expect(parentNode.children).to.have.lengthOf(1);
        expect(parentNode.children[0]).to.equal(childNode);
        expect(childNode.parent).to.equal(parentNode);
    });

    // Test setting parent
    it("should correctly set the parent of a node", () => {
        const parentNode = new InputNode(typename, "parentId", action);
        const childNode = new InputNode(typename, "childId", action);

        childNode.parent = parentNode;

        expect(childNode.parent).to.equal(parentNode);
    });
});
