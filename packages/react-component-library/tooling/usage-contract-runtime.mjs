import React from "react";

const warned = new Set();
const isRecord = (value) =>
  value !== null && typeof value === "object" && !Array.isArray(value);
const describe = (value) => {
  if (value === null) return "null";
  if (Array.isArray(value)) return `array(${value.length})`;
  if (React.isValidElement(value))
    return `element(${componentName(value.type)})`;
  return typeof value;
};

function componentName(type) {
  if (typeof type === "string") return type;
  return (
    type?.displayName ||
    type?.name ||
    type?.render?.displayName ||
    type?.render?.name ||
    "anonymous component"
  );
}

function propertyValue(value, path) {
  let current = [value];
  for (const part of path.split(".")) {
    const spread = part.endsWith("[]");
    const key = spread ? part.slice(0, -2) : part;
    current = current.flatMap((item) => {
      const next = item?.[key];
      if (!spread) return [next];
      return Array.isArray(next) ? next : [];
    });
  }
  return current.length === 1 ? current[0] : current;
}

function matchesType(value, expected) {
  if (expected === "array") return Array.isArray(value);
  if (expected === "object") return isRecord(value);
  return typeof value === expected;
}

function graphemeCount(value) {
  if (typeof Intl.Segmenter === "function") {
    return [
      ...new Intl.Segmenter(undefined, { granularity: "grapheme" }).segment(
        value,
      ),
    ].length;
  }
  return Array.from(value).length;
}

function rclChildren(value) {
  return React.Children.toArray(value).flatMap((child) => {
    if (!React.isValidElement(child)) return [];
    if (child.type === React.Fragment) return rclChildren(child.props.children);
    const name = child.type?.rclUsageComponentName;
    return name ? [{ child, name }] : [];
  });
}

function childRuleViolations(rule, value, field) {
  if (!rule) return [];
  const excluded = new Set(rule.excludedChildComponents ?? []);
  const children = rclChildren(value).filter(({ name }) => !excluded.has(name));
  const violations = [];
  if (
    Number.isInteger(rule.maximumCount) &&
    children.length > rule.maximumCount
  ) {
    violations.push({
      field,
      limit: `maximumCount <= ${rule.maximumCount}`,
      actual: children.length,
    });
  }
  const allowed = rule.allowedChildComponents ?? rule.allowedChildren ?? [];
  if (
    allowed.length > 0 &&
    !allowed.includes("*") &&
    !allowed.includes("any-content-composition")
  ) {
    for (const { name } of children) {
      if (!allowed.includes(name)) {
        violations.push({
          field: `${field}.${name}`,
          limit: `allowedChildComponents = ${allowed.join(", ")}`,
          actual: name,
        });
      }
    }
  }
  return violations;
}

function childrenViolations(contract, props) {
  return childRuleViolations(contract.children, props.children, "children");
}

function slotViolations(contract, props) {
  const violations = [];
  for (const [slot, rule] of Object.entries(contract.slots ?? {})) {
    violations.push(...childRuleViolations(rule, props[slot], `slots.${slot}`));
  }
  return violations;
}

function itemViolations(contract, props) {
  const violations = [];
  const rules =
    contract.collections ??
    contract.itemSets ??
    (contract.items ? [contract.items] : []);
  const legacyFlat =
    rules.length === 0 &&
    contract.property &&
    Number.isInteger(contract.maximumCount)
      ? [contract]
      : rules;
  for (const [ruleIndex, rule] of legacyFlat.entries()) {
    const field = rule.property || `items[${ruleIndex}]`;
    const value = propertyValue(props, rule.property || "items");
    if (rule.identityCallbackProp) {
      const identityCallback = props[rule.identityCallbackProp];
      if (typeof identityCallback !== "function") {
        violations.push({
          field: rule.identityCallbackProp,
          limit: "type function",
          actual:
            identityCallback === undefined
              ? "missing"
              : typeof identityCallback,
        });
      }
    }
    if (value === undefined || value === null) continue;
    if (!Array.isArray(value)) {
      violations.push({
        field,
        limit: "array of item objects",
        actual: describe(value),
      });
      continue;
    }
    if (
      Number.isInteger(rule.maximumCount) &&
      value.length > rule.maximumCount
    ) {
      violations.push({
        field,
        limit: `maximumCount <= ${rule.maximumCount}`,
        actual: value.length,
      });
    }
    for (const [index, item] of value.entries()) {
      if (!isRecord(item)) {
        if (
          (rule.requiredFields?.length ?? 0) > 0 ||
          Object.keys(rule.requiredFieldTypes ?? {}).length > 0
        ) {
          violations.push({
            field: `${field}[${index}]`,
            limit: "object item",
            actual: describe(item),
          });
        }
        continue;
      }
      for (const required of rule.requiredFields ?? []) {
        if (
          propertyValue(item, required) === undefined ||
          propertyValue(item, required) === null
        ) {
          violations.push({
            field: `${field}[${index}].${required}`,
            limit: "required field",
            actual: "missing",
          });
        }
      }
      for (const [required, expectedType] of Object.entries(
        rule.requiredFieldTypes ?? {},
      )) {
        const actual = propertyValue(item, required);
        if (actual === undefined || actual === null) {
          if (!(rule.requiredFields ?? []).includes(required)) {
            violations.push({
              field: `${field}[${index}].${required}`,
              limit: `type ${expectedType}`,
              actual: "missing",
            });
          }
          continue; // requiredFields owns missing-field diagnostics when both are declared.
        }
        if (!matchesType(actual, expectedType)) {
          violations.push({
            field: `${field}[${index}].${required}`,
            limit: `type ${expectedType}`,
            actual: typeof actual,
          });
        }
      }
      if (rule.stableIdField && /^[A-Za-z_$][\w$]*$/.test(rule.stableIdField)) {
        const id = item[rule.stableIdField];
        if (id === undefined || id === null || id === "") {
          violations.push({
            field: `${field}[${index}].${rule.stableIdField}`,
            limit: "stable non-empty identifier",
            actual: id == null ? "missing" : "empty string",
          });
        }
      }
      const labelField = rule.phoneLabelField ?? rule.labelField;
      if (labelField && Number.isInteger(rule.maximumLabelCharacters)) {
        let label = propertyValue(item, labelField);
        if (label === undefined && rule.phoneLabelField && rule.labelField)
          label = propertyValue(item, rule.labelField);
        for (const candidate of Array.isArray(label) ? label : [label]) {
          if (
            candidate !== undefined &&
            candidate !== null &&
            typeof candidate !== "string"
          ) {
            violations.push({
              field: `${field}[${index}].${labelField}`,
              limit: "string label",
              actual: typeof candidate,
            });
          } else if (typeof candidate === "string") {
            const length = graphemeCount(candidate);
            if (length > rule.maximumLabelCharacters) {
              violations.push({
                field: `${field}[${index}].${labelField}`,
                limit: `maximumLabelCharacters <= ${rule.maximumLabelCharacters}`,
                actual: length,
              });
            }
          }
        }
      }
    }
    if (Number.isInteger(rule.mobileMaximumCount)) {
      const mobileField = rule.mobileField ?? "mobile";
      const mobileCount = value.filter(
        (item) => !isRecord(item) || propertyValue(item, mobileField) !== false,
      ).length;
      if (mobileCount > rule.mobileMaximumCount) {
        violations.push({
          field: `${field}.${mobileField}`,
          limit: `mobileMaximumCount <= ${rule.mobileMaximumCount}`,
          actual: mobileCount,
        });
      }
    }
    if (rule.primaryField && Number.isInteger(rule.primaryCount)) {
      const primaryCount = value.filter(
        (item) => isRecord(item) && item[rule.primaryField] === true,
      ).length;
      // `primaryAction: "optional"` means at most N, matching the action-slot
      // evaluator. Only a required primary asserts an exact count. Treating
      // "optional" as exact flagged every zero-primary collection.
      const violates =
        rule.primaryAction === "optional"
          ? primaryCount > rule.primaryCount
          : primaryCount !== rule.primaryCount;
      if (violates) {
        violations.push({
          field: `${field}.${rule.primaryField}`,
          limit:
            rule.primaryAction === "optional"
              ? `primaryCount <= ${rule.primaryCount}`
              : `primaryCount = ${rule.primaryCount}`,
          actual: primaryCount,
        });
      }
    }
  }
  return violations;
}

function parseActionSlot(description) {
  const match =
    typeof description === "string" &&
    description.match(/^\s*([A-Za-z_$][\w$]*)\s*\/\s*([A-Za-z_$][\w$]*)\s*$/);
  return match ? { label: match[1], callback: match[2] } : null;
}

function actionElements(value) {
  return React.Children.toArray(value).flatMap((element) => {
    if (!React.isValidElement(element)) return [];
    if (element.type === React.Fragment)
      return actionElements(element.props.children);
    const component = element.type?.rclUsageComponentName;
    return component || element.type === "button" || element.type === "a"
      ? [element]
      : [];
  });
}

function reactText(value) {
  return React.Children.toArray(value).flatMap((child) => {
    if (typeof child === "string" || typeof child === "number")
      return [String(child)];
    if (React.isValidElement(child)) return reactText(child.props.children);
    return [];
  });
}

function actionViolations(contract, props) {
  const rule = contract.actions;
  if (!rule) return [];
  const violations = [];
  const primary = parseActionSlot(rule.primarySlot);
  const secondary = [
    ...(rule.secondarySlot ? [rule.secondarySlot] : []),
    ...(rule.secondarySlots ?? []),
  ]
    .map(parseActionSlot)
    .filter(Boolean);
  const slots = [
    ...(primary ? [{ ...primary, primary: true }] : []),
    ...secondary.map((slot) => ({ ...slot, primary: false })),
  ];
  let actionCount = 0;
  let primaryCount = 0;
  for (const slot of slots) {
    const callback = props[slot.callback];
    const hasCallback = callback !== undefined && callback !== null;
    const label = props[slot.label];
    if (hasCallback && typeof callback !== "function") {
      violations.push({
        field: slot.callback,
        limit: "type function",
        actual: typeof callback,
      });
    }
    if (hasCallback) {
      actionCount += 1;
      if (slot.primary && typeof callback === "function") primaryCount += 1;
    }
    if (label !== undefined && label !== null) {
      if (typeof label !== "string") {
        violations.push({
          field: slot.label,
          limit: "type string",
          actual: typeof label,
        });
      } else {
        const length = graphemeCount(label);
        if (
          Number.isInteger(rule.maximumLabelCharacters) &&
          length > rule.maximumLabelCharacters
        ) {
          violations.push({
            field: slot.label,
            limit: `maximumLabelCharacters <= ${rule.maximumLabelCharacters}`,
            actual: length,
          });
        }
      }
    }
  }
  const recovery = parseActionSlot(rule.recoveryAction);
  if (recovery) {
    const callback = props[recovery.callback];
    const label = props[recovery.label];
    if (
      callback !== undefined &&
      callback !== null &&
      typeof callback !== "function"
    ) {
      violations.push({
        field: recovery.callback,
        limit: "type function",
        actual: typeof callback,
      });
    }
    if (label !== undefined && label !== null && typeof label !== "string") {
      violations.push({
        field: recovery.label,
        limit: "type string",
        actual: typeof label,
      });
    } else if (
      typeof label === "string" &&
      Number.isInteger(rule.maximumLabelCharacters)
    ) {
      const length = graphemeCount(label);
      if (length > rule.maximumLabelCharacters) {
        violations.push({
          field: recovery.label,
          limit: `maximumLabelCharacters <= ${rule.maximumLabelCharacters}`,
          actual: length,
        });
      }
    }
  }
  // Older action contracts refer to named ReactNode props. Count only known
  // action elements, so arbitrary explanatory markup does not spend the cap.
  if (
    slots.length === 0 &&
    rule.primarySlot &&
    /^[A-Za-z_$][\w$]*$/.test(rule.primarySlot)
  ) {
    const primaryElements = actionElements(props[rule.primarySlot]);
    const secondaryName =
      typeof rule.secondarySlot === "string" &&
      /^[A-Za-z_$][\w$]*$/.test(rule.secondarySlot)
        ? rule.secondarySlot
        : undefined;
    const secondaryElements = secondaryName
      ? actionElements(props[secondaryName])
      : [];
    actionCount = primaryElements.length + secondaryElements.length;
    primaryCount = primaryElements.length;
    for (const element of [...primaryElements, ...secondaryElements]) {
      const label =
        typeof element.props?.label === "string"
          ? element.props.label
          : reactText(element.props?.children).join("");
      const length = graphemeCount(label);
      if (
        Number.isInteger(rule.maximumLabelCharacters) &&
        length > rule.maximumLabelCharacters
      ) {
        violations.push({
          field: `${rule.primarySlot}/${secondaryName ?? "actions"}.label`,
          limit: `maximumLabelCharacters <= ${rule.maximumLabelCharacters}`,
          actual: length,
        });
      }
    }
  }
  if (Number.isInteger(rule.maximumCount) && actionCount > rule.maximumCount) {
    violations.push({
      field: "actions",
      limit: `maximumCount <= ${rule.maximumCount}`,
      actual: actionCount,
    });
  }
  const primaryLimit = Number.isInteger(rule.primaryCount)
    ? rule.primaryCount
    : 1;
  if (rule.primaryAction === "required" && primaryCount !== primaryLimit) {
    violations.push({
      field: "actions.primary",
      limit: `primaryCount = ${primaryLimit}`,
      actual: primaryCount,
    });
  } else if (rule.primaryAction === "optional" && primaryCount > primaryLimit) {
    violations.push({
      field: "actions.primary",
      limit: `primaryCount <= ${primaryLimit}`,
      actual: primaryCount,
    });
  }
  return violations;
}

export function inspectUsageContract(contract, props) {
  return [
    ...childrenViolations(contract, props),
    ...slotViolations(contract, props),
    ...itemViolations(contract, props),
    ...actionViolations(contract, props),
  ];
}

function callSite() {
  const owner =
    React.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED?.ReactCurrentOwner
      ?.current;
  const source = owner?._debugSource ?? owner?._debugOwner?._debugSource;
  if (source?.fileName) {
    return `${source.fileName}:${source.lineNumber ?? 1}:${source.columnNumber ?? 1}`;
  }
  const frame = new Error().stack
    ?.split("\n")
    .find(
      (line) =>
        line.includes(" at ") &&
        !line.includes("usage-contract-runtime") &&
        !line.includes("node_modules/react"),
    );
  return (
    frame?.trim() ??
    "call site unavailable (React DEV source metadata is required)"
  );
}

export function wrapUsageContract(Component, contract) {
  if (typeof Component !== "function" && typeof Component !== "object")
    return Component;
  const identity = `${contract.componentId}@${contract.version}`;
  const Wrapped = React.forwardRef((props, ref) => {
    for (const violation of inspectUsageContract(contract, props)) {
      const location = callSite();
      const signature = `${identity}|${location}|${violation.field}|${violation.limit}|${violation.actual}`;
      if (warned.has(signature)) continue;
      warned.add(signature);
      console.warn(
        `[RCL usage contract] ${identity}: ${violation.field} violates ${violation.limit}; received ${JSON.stringify(violation.actual)} at ${location}`,
      );
    }
    const supportsRef =
      Component?.$$typeof === Symbol.for("react.forward_ref") ||
      (Component?.$$typeof === Symbol.for("react.memo") &&
        Component.type?.$$typeof === Symbol.for("react.forward_ref"));
    return React.createElement(
      Component,
      supportsRef ? { ...props, ref } : props,
    );
  });
  Wrapped.rclUsageComponentName = contract.componentId.split(":").at(-1);
  Wrapped.displayName = `${contract.componentId.split(":").at(-1)}ContractDEV`;
  return Wrapped;
}
