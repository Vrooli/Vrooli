import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const overlayNames = new Set(["Dialog", "BottomSheet", "FullPageDrawer", "DrawerShell"]);
const sharedBar = "OverlayActionBar";
const libraryRoot = path.join(repo, "scenarios/react-component-library/library/components");
const actionBarContract = JSON.parse(
  fs.readFileSync(
    path.join(libraryRoot, "OverlayActionBar/versions/1.0.0/usage-contract.json"),
    "utf8",
  ),
);
const contractsByComponent = new Map();
for (const name of fs.readdirSync(libraryRoot)) {
  const componentDir = path.join(libraryRoot, name);
  const metadataPath = path.join(componentDir, "component.json");
  if (!fs.existsSync(metadataPath)) continue;
  try {
    const metadata = JSON.parse(fs.readFileSync(metadataPath, "utf8"));
    const contractPath = path.join(
      componentDir,
      "versions",
      metadata.latest,
      "usage-contract.json",
    );
    if (fs.existsSync(contractPath))
      contractsByComponent.set(name, JSON.parse(fs.readFileSync(contractPath, "utf8")));
  } catch {
    /* The catalog checker reports malformed declarations with component context. */
  }
}
const elements = (node) => {
  const found = [];
  const visit = (child) => {
    if (!child || typeof child !== "object") return;
    if (
      child.type === "JSXElement" ||
      child.type === "JSXOpeningElement" ||
      child.type === "JSXClosingElement"
    )
      found.push(child);
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
  return name?.type === "JSXIdentifier"
    ? name.name
    : name?.type === "JSXMemberExpression"
      ? name.property?.name
      : "";
};
const unwrapExpression = (node) => {
  let current = node;
  while (
    current &&
    [
      "TSAsExpression",
      "TSTypeAssertion",
      "TSNonNullExpression",
      "ParenthesizedExpression",
      "ChainExpression",
    ].includes(current.type)
  )
    current = current.expression;
  return current;
};
const staticString = (node, declarations, seen = new Set()) => {
  node = unwrapExpression(node);
  if (node?.type === "Literal" && typeof node.value === "string") return node.value;
  if (node?.type === "TemplateLiteral" && node.expressions.length === 0)
    return node.quasis[0]?.value?.cooked;
  if (node?.type === "Identifier" && !seen.has(node.name)) {
    seen.add(node.name);
    return staticString(declarations.get(node.name), declarations, seen);
  }
  return undefined;
};
const looksLikeStableID = (node) => {
  node = unwrapExpression(node);
  if (node?.type === "Identifier") return true;
  if (node?.type === "MemberExpression" && !node.computed && node.property?.type === "Identifier")
    return true;
  if (node?.type === "TemplateLiteral" && node.expressions.length > 0)
    return node.expressions.every(looksLikeStableID);
  if (node?.type !== "CallExpression" || node.arguments?.length !== 1) return false;
  const callee = node.callee;
  const recognizedStringifier = callee?.type === "Identifier" && callee.name === "String";
  const recognizedTranslator =
    (callee?.type === "Identifier" && ["t", "translate", "strings"].includes(callee.name)) ||
    (callee?.type === "MemberExpression" &&
      !callee.computed &&
      callee.property?.name === "t" &&
      callee.object?.type === "Identifier" &&
      ["i18n", "strings"].includes(callee.object.name));
  if (!recognizedStringifier && !recognizedTranslator) return false;
  if (recognizedTranslator) {
    const key = unwrapExpression(node.arguments[0]);
    return (
      (key?.type === "Literal" && typeof key.value === "string" && key.value.trim().length > 0) ||
      key?.type === "Identifier" ||
      (key?.type === "MemberExpression" &&
        !key.computed &&
        key.property?.type === "Identifier")
    );
  }
  return (
    node.arguments[0]?.type === "Identifier" ||
    (node.arguments[0]?.type === "MemberExpression" &&
      !node.arguments[0].computed &&
      node.arguments[0].property?.type === "Identifier")
  );
};
const isTranslationCall = (node) => {
  if (node?.type !== "CallExpression") return false;
  const callee = node.callee;
  return (
    (callee?.type === "Identifier" && ["t", "translate", "strings"].includes(callee.name)) ||
    (callee?.type === "MemberExpression" &&
      !callee.computed &&
      callee.property?.name === "t" &&
      callee.object?.type === "Identifier" &&
      ["i18n", "strings"].includes(callee.object.name))
  );
};
const stringLiterals = (node) => {
  const values = [];
  const collect = (item) => {
    if (!item || typeof item !== "object") return;
    if (item.type === "Literal" && typeof item.value === "string") {
      values.push(item.value);
      return;
    }
    if (item.type === "TemplateLiteral") {
      const staticText = (item.quasis ?? [])
        .map((quasi) => quasi.value?.cooked ?? quasi.value?.raw ?? "")
        .join("");
      if (staticText.trim() || (item.expressions ?? []).length === 0)
        values.push(staticText);
      return;
    }
    // Translation keys and interpolation arguments are source identifiers, not
    // rendered labels. Runtime/story checks own localized length and content.
    if (isTranslationCall(item)) return;
    for (const [key, value] of Object.entries(item)) {
      if (key === "parent") continue;
      if (Array.isArray(value)) value.forEach(collect);
      else if (value && typeof value === "object") collect(value);
    }
  };
  collect(node);
  return values;
};
const propertyValue = (object, name) =>
  object?.type === "ObjectExpression"
    ? object.properties.find(
        (property) =>
          (property.type === "Property" || property.type === "ObjectProperty") &&
          (property.key?.name ?? property.key?.value) === name,
      )?.value
    : undefined;
const objectAlternatives = (node) => {
  node = unwrapExpression(node);
  if (node?.type === "ObjectExpression") return [node];
  if (node?.type === "ConditionalExpression")
    return [...objectAlternatives(node.consequent), ...objectAlternatives(node.alternate)];
  return [];
};
const propertyValues = (object, name) =>
  objectAlternatives(object).map((alternative) => propertyValue(alternative, name));

function arrayFromExpression(expression, declarations, seen = new WeakSet()) {
  if (!expression) return null;
  if (typeof expression === "object") {
    if (seen.has(expression)) return null;
    seen.add(expression);
  }
  if (expression.type === "ArrayExpression") {
    const elements = [];
    let uncertain = false;
    for (const item of expression.elements) {
      if (!item || item.type !== "SpreadElement") {
        if (item) elements.push(item);
        continue;
      }
      const spread = arrayFromExpression(item.argument, declarations, seen);
      if (!spread) {
        // A dynamic spread cannot be counted or inspected safely by syntax
        // alone. Preserve findings for the known entries and leave the rest
        // to the generated type/runtime contract instead of inventing missing
        // fields on the SpreadElement node.
        uncertain = true;
        continue;
      }
      elements.push(...spread.elements);
      uncertain ||= spread.uncertain;
    }
    return { node: expression, elements, count: elements.length, uncertain };
  }
  if (expression.type === "Identifier")
    return arrayFromExpression(declarations.get(expression.name), declarations, seen);
  if (expression.type === "ConditionalExpression") {
    const consequent = arrayFromExpression(expression.consequent, declarations, seen);
    const alternate = arrayFromExpression(expression.alternate, declarations, seen);
    if (!consequent && !alternate) return null;
    const branches = [consequent, alternate].filter(Boolean);
    return {
      node: expression,
      elements: branches.flatMap((branch) => branch.elements),
      count: Math.max(0, ...branches.map((branch) => branch.count)),
      uncertain: branches.some((branch) => branch.uncertain),
    };
  }
  if (
    [
      "TSAsExpression",
      "TSTypeAssertion",
      "TSNonNullExpression",
      "ParenthesizedExpression",
    ].includes(expression.type)
  ) {
    return arrayFromExpression(expression.expression, declarations, seen);
  }
  if (
    expression.type === "CallExpression" &&
    expression.callee?.type === "MemberExpression" &&
    !expression.callee.computed &&
    expression.callee.property?.name === "map"
  ) {
    const input = arrayFromExpression(expression.callee.object, declarations, seen);
    const mapper = expression.arguments?.[0];
    if (!input || !mapper) return null;
    const mapped =
      mapper.body?.type === "BlockStatement"
        ? mapper.body.body.find((statement) => statement.type === "ReturnStatement")?.argument
        : mapper.body;
    if (!mapped) return null;
    return {
      node: expression,
      elements: input.elements.map(() => mapped),
      count: input.count,
      uncertain: input.uncertain,
    };
  }
  return null;
}

export default {
  meta: {
    type: "problem",
    docs: {
      description: "enforce RCL usage contracts and shared overlay parts in scenario UI call sites",
    },
    schema: [],
    messages: {
      tooMany: "{{component}} has {{count}} actions; contract maximum is {{maximum}}.",
      missingField: "{{component}} action {{index}} must include required field '{{field}}'.",
      invalidFieldType: "{{component}} action {{index}} field '{{field}}' must be a {{type}}.",
      unstableID: "{{component}} action {{index}} id must be a static, stable string.",
      emptyLabel: "{{component}} action {{index}} label must be a non-empty string.",
      longLabel:
        "{{component}} action '{{id}}' has a {{length}} character label; contract maximum is {{maximum}}.",
      primaryCount:
        "{{component}} actions must declare {{required}} primary action(s); found {{count}}.",
      rebuild:
        "Overlay footer renders buttons directly; use the shared OverlayActionBar through the actions contract.",
      customRebuild:
        "Custom {{name}} renders buttons without OverlayActionBar; compose the shared action part instead.",
      tooManyChildren: "{{component}} has {{count}} child nodes; contract maximum is {{maximum}}.",
      disallowedChild:
        "{{component}} does not allow child component '{{child}}'; allowed children: {{allowed}}.",
    },
  },
  create(context) {
    const declarations = new Map();
    const componentDefinitions = new Map();
    const importedComponents = new Map();
    const overlayFooterBindings = new Set();
    const source = context.sourceCode ?? context.getSourceCode();
    const report = (node, messageId, data = {}) => context.report({ node, messageId, data });
    const catalogName = (name) =>
      importedComponents.has(name)
        ? importedComponents.get(name)
        : contractsByComponent.has(name)
          ? name
          : null;
    const hasShared = (node) =>
      elements(node).some(
        (item) =>
          (item.type === "JSXElement" && jsxName(item) === sharedBar) ||
          (item.type === "JSXOpeningElement" && jsxName(item) === sharedBar),
      );
    const hasButtons = (node) =>
      elements(node).filter(
        (item) =>
          (item.type === "JSXElement" && jsxName(item) === "button") ||
          (item.type === "JSXOpeningElement" && jsxName(item) === "button"),
      ).length > 0;
    const hasActionControls = (node) =>
      elements(node).some((item) => {
        const name = jsxName(item);
        if (name === "button" || /^Button$|ActionButton$/.test(name)) return true;
        const opening = item.type === "JSXElement" ? item.openingElement : item;
        return (opening?.attributes ?? []).some(
          (attribute) =>
            attribute.type === "JSXAttribute" &&
            attribute.name?.name === "role" &&
            attribute.value?.value === "button",
        );
      });
    const isSharedPartName = (name) =>
      !/header|toolbar/i.test(name) &&
      /^(?:dialog|drawer|sheet|overlay).*(?:footer|actions|actionbar)$/i.test(name);
    const checkCollection = (component, property, node, contract, openingElement) => {
      const expression = node?.type === "JSXExpressionContainer" ? node.expression : null;
      const array = arrayFromExpression(expression, declarations);
      if (!array) return;
      const actions = array.elements.filter(Boolean);
      if (array.count > contract.maximumCount)
        report(array.node, "tooMany", {
          component,
          count: array.count,
          maximum: contract.maximumCount,
        });
      if (contract.identityCallbackProp) {
        const callback = (openingElement?.attributes ?? []).find(
          (attribute) =>
            attribute.type === "JSXAttribute" &&
            attribute.name?.name === contract.identityCallbackProp,
        );
        if (!callback)
          report(node, "missingField", {
            component: `${component}.${property}`,
            index: 1,
            field: contract.identityCallbackProp,
          });
      }
      if (contract.itemShape === "component-owned") return;
      let primaries = 0;
      for (const [index, action] of actions.entries()) {
        const alternatives = objectAlternatives(action);
        if (alternatives.length === 0) continue;
        const idField =
          contract.stableIdField ??
          (contract.requiredFields ?? []).find((field) => field === "id" || field === "value") ??
          "id";
        const requiredLabelField =
          contract.labelField ??
          (contract.requiredFields ?? []).find((field) => field === "label" || field === "title") ??
          "label";
        const labelField = contract.phoneLabelField ?? requiredLabelField;
        const idNodes = alternatives.map((alternative) => propertyValue(alternative, idField));
        const labelNodes = alternatives.map(
          (alternative) =>
            propertyValue(alternative, labelField) ??
            propertyValue(alternative, requiredLabelField),
        );
        const id = idNodes.map((idNode) => staticString(idNode, declarations)).find((value) => value !== undefined);
        for (const field of contract.requiredFields ?? []) {
          const fieldNodes = alternatives.map((alternative) => propertyValue(alternative, field));
          if (fieldNodes.some((fieldNode) => !fieldNode))
            report(action, "missingField", {
              component: `${component}.${property}`,
              index: index + 1,
              field,
            });
          const expectedType = contract.requiredFieldTypes?.[field];
          const wrongNode = fieldNodes.find((fieldNode) =>
            expectedType === "string"
              ? fieldNode?.type === "Literal" && typeof fieldNode.value !== "string"
              : expectedType === "function" &&
                fieldNode &&
                !["ArrowFunctionExpression", "FunctionExpression", "Identifier"].includes(
                  fieldNode.type,
                ),
          );
          if (wrongNode)
            report(wrongNode, "invalidFieldType", {
              component: `${component}.${property}`,
              index: index + 1,
              field,
              type: expectedType,
            });
        }
        if (idNodes.some((idNode) => !idNode))
          report(action, "missingField", {
            component: `${component}.${property}`,
            index: index + 1,
            field: idField,
          });
        else if (
          idNodes.some((idNode) => {
            const value = staticString(idNode, declarations);
            if (typeof value === "string")
              return value.length === 0 && !(component === "Select" && idField === "value");
            return !looksLikeStableID(idNode);
          })
        )
          report(action, "unstableID", { component: `${component}.${property}`, index: index + 1 });
        const labels = labelNodes.flatMap((labelNode) => stringLiterals(labelNode));
        if (labelNodes.some((labelNode) => !labelNode) || labels.some((label) => !label.trim()))
          report(action, "emptyLabel", { component: `${component}.${property}`, index: index + 1 });
        const maximumLabelCharacters = contract.maximumLabelCharacters ?? contract.labelMaxChars;
        const length = Math.max(0, ...labels.map((label) => Array.from(label).length));
        if (length > maximumLabelCharacters)
          report(action, "longLabel", {
            component,
            id: id ?? `#${index + 1}`,
            length,
            maximum: maximumLabelCharacters,
          });
        if (contract.primaryField) {
          const branchPrimaryCount = Math.max(
            0,
            ...alternatives.map(
              (alternative) =>
                propertyValue(alternative, contract.primaryField)?.value === true ? 1 : 0,
            ),
          );
          primaries += branchPrimaryCount;
        }
      }
      if (contract.mobileMaximumCount !== undefined) {
        const mobileField = contract.mobileField ?? "mobile";
        const mobileCount = actions.filter((action) => {
          const alternatives = objectAlternatives(action);
          // Unknown expressions remain conservatively phone-visible. Do not
          // let a dynamic item erase itself from the mobile budget.
          return (
            alternatives.length === 0 ||
            alternatives.some(
              (alternative) => propertyValue(alternative, mobileField)?.value !== false,
            )
          );
        }).length;
        if (mobileCount > contract.mobileMaximumCount)
          report(array.node, "tooMany", {
            component,
            count: mobileCount,
            maximum: contract.mobileMaximumCount,
          });
      }
      if (contract.primaryField && primaries !== contract.primaryCount)
        report(array.node, "primaryCount", {
          component: `${component}.${property}`,
          required: contract.primaryCount,
          count: primaries,
        });
    };
    return {
      Program(node) {
        walk(node, (child) => {
          if (child.type === "ImportDeclaration") {
            const source = child.source?.value;
            for (const specifier of child.specifiers ?? []) {
              if (specifier.type !== "ImportSpecifier") continue;
              const local = specifier.local?.name;
              const imported = specifier.imported?.name ?? specifier.imported?.value;
              if (!local) continue;
              importedComponents.set(
                local,
                typeof source === "string" &&
                  source.startsWith("@vrooli/react-component-library/") &&
                  contractsByComponent.has(imported)
                  ? imported
                  : null,
              );
            }
          }
          if (child.type === "VariableDeclarator" && child.id?.type === "Identifier")
            declarations.set(child.id.name, child.init);
          if (child.type === "FunctionDeclaration" && child.id?.name)
            componentDefinitions.set(child.id.name, child);
          if (child.type === "VariableDeclarator" && child.id?.type === "Identifier")
            componentDefinitions.set(child.id.name, child.init);
        });
      },
      "Program:exit"(node) {
        walk(node, (child) => {
          if (
            child.type === "VariableDeclarator" &&
            overlayFooterBindings.has(child.id?.name) &&
            hasButtons(child.init) &&
            !hasShared(child.init)
          ) {
            report(child, "customRebuild", { name: "overlay footer" });
          }
        });
      },
      JSXOpeningElement(node) {
        const component = catalogName(jsxName(node));
        if (!component) return;
        const declared = contractsByComponent.get(component);
        if (declared) {
          const collections = [
            ...(declared.items ? [declared.items] : []),
            ...(declared.collections ?? []),
          ];
          for (const attribute of node.attributes ?? []) {
            if (attribute.type !== "JSXAttribute") continue;
            const property = attribute.name?.name;
            const contract = collections.find((item) => item.property === property);
            if (contract) checkCollection(component, property, attribute.value, contract, node);
          }
        }
        if (!overlayNames.has(component)) return;
        for (const attribute of node.attributes ?? []) {
          if (attribute.type !== "JSXAttribute") continue;
          if (attribute.name?.name === "actions")
            checkCollection(component, "actions", attribute.value, actionBarContract.items, node);
          if (
            attribute.name?.name === "footer" &&
            attribute.value?.type === "JSXExpressionContainer"
          ) {
            const footer = attribute.value.expression;
            if (footer.type === "Identifier") overlayFooterBindings.add(footer.name);
            const nestedComponents = elements(footer)
              .filter((item) => item.type === "JSXOpeningElement")
              .map(jsxName)
              .filter((name) => /^[A-Z]/.test(name));
            const nestedRebuild = nestedComponents.some((name) => {
              const definition = componentDefinitions.get(name);
              return definition && hasButtons(definition) && !hasShared(definition);
            });
            if (hasButtons(footer) && !hasShared(footer)) report(attribute, "rebuild");
            else if (nestedRebuild)
              report(attribute, "customRebuild", { name: "overlay footer component" });
          }
        }
      },
      JSXElement(node) {
        const rawName = jsxName(node);
        const component = catalogName(rawName);
        const childContract = contractsByComponent.get(component)?.children;
        if (childContract) {
          // The contract allowlist names RCL children. Text nodes, HTML
          // wrappers and icons from another library are content, not RCL
          // component slots, and must not consume the component budget.
          const excludedChildren = new Set(childContract.excludedChildComponents ?? []);
          const children = (node.children ?? []).filter(
            (child) =>
              child.type === "JSXElement" &&
              catalogName(jsxName(child)) &&
              !excludedChildren.has(jsxName(child)),
          );
          if (children.length > childContract.maximumCount)
            report(node, "tooManyChildren", {
              component,
              count: children.length,
              maximum: childContract.maximumCount,
            });
          const allowed =
            childContract.allowedChildComponents ?? childContract.allowedChildren ?? [];
          const permitsComposition =
            allowed.includes("*") || allowed.includes("any-content-composition");
          if (!permitsComposition)
            for (const child of children) {
              if (child.type !== "JSXElement") continue;
              const childName = catalogName(jsxName(child));
              if (!childName || allowed.includes(childName)) continue;
              report(child, "disallowedChild", {
                component,
                child: childName,
                allowed: allowed.join(", ") || "none",
              });
            }
        }
        const footerLike = jsxName(node) === "footer" || /Footer$/.test(jsxName(node));
        if (!footerLike || !hasActionControls(node) || hasShared(node)) return;
        let parent = node.parent;
        while (parent) {
          if (parent.type === "JSXElement" && overlayNames.has(jsxName(parent))) {
            report(node, "rebuild");
            return;
          }
          parent = parent.parent;
        }
      },
      FunctionDeclaration(node) {
        const name = node.id?.name ?? "";
        if (isSharedPartName(name) && hasButtons(node) && !hasShared(node))
          report(node, "customRebuild", { name });
      },
      VariableDeclarator(node) {
        const name = node.id?.name ?? "";
        if (isSharedPartName(name) && hasButtons(node.init) && !hasShared(node.init))
          report(node, "customRebuild", { name });
      },
    };
  },
};
