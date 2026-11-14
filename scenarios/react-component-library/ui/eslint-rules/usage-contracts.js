import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const contractPath = path.join(repo, "scenarios/react-component-library/library/components/OverlayActionBar/versions/1.0.0/usage-contract.json");
const actionContract = JSON.parse(fs.readFileSync(contractPath, "utf8"));
const overlayNames = new Set(["Dialog", "BottomSheet", "FullPageDrawer", "DrawerShell"]);
const sharedBar = "OverlayActionBar";
const elements = (node) => {
  const found = [];
  const visit = (child) => {
    if (!child || typeof child !== "object") return;
    if (child.type === "JSXElement" || child.type === "JSXOpeningElement" || child.type === "JSXClosingElement") found.push(child);
    for (const [key, value] of Object.entries(child)) {
      if (key === "parent") continue;
      if (Array.isArray(value)) value.forEach(visit);
      else if (value && typeof value === "object") visit(value);
    }
  };
  visit(node);
  return found;
};
const walk = (node, visitor) => {
  if (!node || typeof node !== "object") return;
  visitor(node);
  for (const [key, value] of Object.entries(node)) {
    if (key === "parent") continue;
    if (Array.isArray(value)) value.forEach((child) => walk(child, visitor));
    else if (value && typeof value === "object") walk(value, visitor);
  }
};
const jsxName = (element) => {
  const opening = element?.type === "JSXElement" ? element.openingElement : element;
  const name = opening?.name;
  return name?.type === "JSXIdentifier" ? name.name : name?.type === "JSXMemberExpression" ? name.property?.name : "";
};
const literalString = (node) => node?.type === "Literal" && typeof node.value === "string" ? node.value : undefined;
const stringLiterals = (node) => {
  const values = [];
  walk(node, (item) => { if (item.type === "Literal" && typeof item.value === "string") values.push(item.value); });
  return values;
};
const propertyValue = (object, name) => object?.type === "ObjectExpression" ? object.properties.find((property) => (property.type === "Property" || property.type === "ObjectProperty") && (property.key?.name ?? property.key?.value) === name)?.value : undefined;
const isStaticString = (node) => node?.type === "Literal" && typeof node.value === "string" || node?.type === "TemplateLiteral" && node.expressions.length === 0;

function arrayFromExpression(expression, declarations) {
  if (!expression) return null;
  if (expression.type === "ArrayExpression") return expression;
  if (expression.type === "Identifier") return arrayFromExpression(declarations.get(expression.name), declarations);
  let result = null;
  for (const child of Object.values(expression)) {
    if (Array.isArray(child)) {
      for (const item of child) { result = arrayFromExpression(item, declarations); if (result) break; }
    } else if (child && typeof child === "object") {
      result = arrayFromExpression(child, declarations);
    }
    if (result) break;
  }
  return result;
}

export default {
  meta: {
    type: "problem",
    docs: { description: "enforce RCL usage contracts and shared overlay parts in scenario UI call sites" },
    schema: [],
    messages: {
      tooMany: "{{component}} has {{count}} actions; contract maximum is {{maximum}}.",
      missingField: "{{component}} action {{index}} must include required field '{{field}}'.",
      invalidFieldType: "{{component}} action {{index}} field '{{field}}' must be a {{type}}.",
      unstableID: "{{component}} action {{index}} id must be a static, stable string.",
      emptyLabel: "{{component}} action {{index}} label must be a non-empty string.",
      longLabel: "{{component}} action '{{id}}' has a {{length}} character label; contract maximum is {{maximum}}.",
      primaryCount: "{{component}} actions must declare {{required}} primary action(s); found {{count}}.",
      rebuild: "Overlay footer renders buttons directly; use the shared OverlayActionBar through the actions contract.",
      customRebuild: "Custom {{name}} renders buttons without OverlayActionBar; compose the shared action part instead.",
    },
  },
  create(context) {
    const declarations = new Map();
    const overlayFooterBindings = new Set();
    const source = context.sourceCode ?? context.getSourceCode();
    const report = (node, messageId, data = {}) => context.report({ node, messageId, data });
    const hasShared = (node) => elements(node).some((item) => item.type === "JSXElement" && jsxName(item) === sharedBar || item.type === "JSXOpeningElement" && jsxName(item) === sharedBar);
    const hasButtons = (node) => elements(node).filter((item) => item.type === "JSXElement" && jsxName(item) === "button" || item.type === "JSXOpeningElement" && jsxName(item) === "button").length > 0;
    const isSharedPartName = (name) => !/header|toolbar/i.test(name) && /^(?:dialog|drawer|sheet|overlay).*(?:footer|actions|actionbar)$/i.test(name);
    const checkActions = (component, node) => {
      const expression = node?.type === "JSXExpressionContainer" ? node.expression : null;
      const array = arrayFromExpression(expression, declarations);
      if (!array) return;
      const actions = array.elements.filter(Boolean);
      if (actions.length > actionContract.items.maximumCount) report(array, "tooMany", { component, count: actions.length, maximum: actionContract.items.maximumCount });
      let primaries = 0;
      for (const [index, action] of actions.entries()) {
        const idNode = propertyValue(action, "id");
        const labelNode = propertyValue(action, "label");
        const id = literalString(idNode) ?? (idNode?.type === "TemplateLiteral" && idNode.expressions.length === 0 ? idNode.quasis[0]?.value?.cooked : undefined);
        for (const field of actionContract.items.requiredFields) {
          const fieldNode = propertyValue(action, field);
          if (!fieldNode) report(action, "missingField", { component, index: index + 1, field });
          const expectedType = actionContract.items.requiredFieldTypes?.[field];
          const obviouslyWrong = expectedType === "string"
            ? fieldNode?.type === "Literal" && typeof fieldNode.value !== "string"
            : expectedType === "function" && fieldNode && !["ArrowFunctionExpression", "FunctionExpression", "Identifier"].includes(fieldNode.type);
          if (obviouslyWrong) report(fieldNode, "invalidFieldType", { component, index: index + 1, field, type: expectedType });
        }
        if (!isStaticString(idNode) || !id?.trim()) report(action, "unstableID", { component, index: index + 1 });
        const labels = stringLiterals(labelNode);
        if (!labelNode || labels.some((label) => !label.trim())) report(action, "emptyLabel", { component, index: index + 1 });
        for (const label of labels) {
          const length = Array.from(label).length;
          if (length > actionContract.items.maximumLabelCharacters) report(action, "longLabel", { component, id: id ?? `#${index + 1}`, length, maximum: actionContract.items.maximumLabelCharacters });
        }
        if (propertyValue(action, actionContract.items.primaryField)?.value === true) primaries++;
      }
      if (primaries !== actionContract.items.primaryCount) report(array, "primaryCount", { component, required: actionContract.items.primaryCount, count: primaries });
    };
    return {
      Program(node) {
        walk(node, (child) => {
          if (child.type === "VariableDeclarator" && child.id?.type === "Identifier") declarations.set(child.id.name, child.init);
        });
      },
      "Program:exit"(node) {
        walk(node, (child) => {
          if (child.type === "VariableDeclarator" && overlayFooterBindings.has(child.id?.name) && hasButtons(child.init) && !hasShared(child.init)) {
            report(child, "customRebuild", { name: "overlay footer" });
          }
        });
      },
      JSXOpeningElement(node) {
        const component = jsxName(node);
        if (!overlayNames.has(component)) return;
        for (const attribute of node.attributes ?? []) {
          if (attribute.type !== "JSXAttribute") continue;
          if (attribute.name?.name === "actions") checkActions(component, attribute.value);
          if (attribute.name?.name === "footer" && attribute.value?.type === "JSXExpressionContainer") {
            if (attribute.value.expression.type === "Identifier") overlayFooterBindings.add(attribute.value.expression.name);
            else if (hasButtons(attribute.value.expression) && !hasShared(attribute.value.expression)) report(attribute, "rebuild");
          }
        }
      },
      JSXElement(node) {
        if (jsxName(node) !== "footer" || !hasButtons(node) || hasShared(node)) return;
        let parent = node.parent;
        while (parent) {
          if (parent.type === "JSXElement" && overlayNames.has(jsxName(parent))) { report(node, "rebuild"); return; }
          parent = parent.parent;
        }
      },
      FunctionDeclaration(node) {
        const name = node.id?.name ?? "";
        if (isSharedPartName(name) && hasButtons(node) && !hasShared(node)) report(node, "customRebuild", { name });
      },
      VariableDeclarator(node) {
        const name = node.id?.name ?? "";
        if (isSharedPartName(name) && hasButtons(node.init) && !hasShared(node.init)) report(node, "customRebuild", { name });
      },
    };
  },
};
