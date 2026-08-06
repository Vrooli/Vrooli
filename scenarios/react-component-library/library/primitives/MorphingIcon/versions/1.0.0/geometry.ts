import {
  ICON_REGISTRY,
  type IconDefinition,
  type IconName,
} from "../../../../foundations/IconRegistry/versions/1.0.0/IconRegistry";

export type Point = { x: number; y: number };
export type NormalizedCommand = { type: "M" | "L"; point: Point };

export interface NormalizedIconGeometry {
  definition: IconDefinition;
  commands: NormalizedCommand[];
  topology: string;
  d: string;
}

const commandPattern = /([a-z])([^a-z]*)/gi;
const numberPattern = /-?(?:\d+\.?\d*|\.\d+)/g;

function numbers(value: string) {
  return (value.match(numberPattern) ?? []).map(Number);
}

function appendPoint(
  commands: NormalizedCommand[],
  type: "M" | "L",
  point: Point,
) {
  commands.push({ type, point });
}

/**
 * Normalize the small, registry-owned SVG vocabulary into absolute move/line
 * commands. Unsupported path commands are conservatively represented by
 * their endpoint, so arbitrary registry additions fall back instead of
 * producing a visually dishonest morph.
 */
export function normalizeIconGeometry(name: IconName): NormalizedIconGeometry {
  const registry = ICON_REGISTRY as Partial<Record<string, IconDefinition>>;
  const definition = registry[name] ?? ICON_REGISTRY.check;
  const commands: NormalizedCommand[] = [];
  let current: Point = { x: 0, y: 0 };

  for (const match of definition.path.matchAll(commandPattern)) {
    const command = match[1];
    if (!command) continue;
    const relative = command === command.toLowerCase();
    const type = command.toUpperCase();
    const values = numbers(match[2] ?? "");
    const readPoint = (offset: number): Point => {
      const x = values[offset] ?? current.x;
      const y = values[offset + 1] ?? current.y;
      return relative ? { x: current.x + x, y: current.y + y } : { x, y };
    };

    if (type === "M" || type === "L" || type === "T") {
      for (let index = 0; index + 1 < values.length; index += 2) {
        const point = readPoint(index);
        appendPoint(commands, type === "M" && index === 0 ? "M" : "L", point);
        current = point;
      }
      continue;
    }

    if (type === "H" || type === "V") {
      for (const value of values) {
        const point =
          type === "H"
            ? { x: relative ? current.x + value : value, y: current.y }
            : { x: current.x, y: relative ? current.y + value : value };
        appendPoint(commands, "L", point);
        current = point;
      }
      continue;
    }

    if (type === "C" || type === "S" || type === "Q" || type === "A") {
      const stride = type === "C" ? 6 : type === "A" ? 7 : type === "Q" ? 4 : 4;
      for (let index = stride - 2; index < values.length; index += stride) {
        const point = readPoint(index);
        appendPoint(commands, "L", point);
        current = point;
      }
      continue;
    }

    if (type === "Z") {
      const first = commands.find((entry) => entry.type === "M")?.point;
      if (first) {
        appendPoint(commands, "L", first);
        current = first;
      }
    }
  }

  return {
    definition,
    commands,
    topology: commands.map(({ type }) => type).join(""),
    d: commands
      .map(({ type, point }) => `${type}${point.x} ${point.y}`)
      .join(" "),
  };
}

export function canMorph(
  from: NormalizedIconGeometry,
  to: NormalizedIconGeometry,
) {
  return (
    from.definition.viewBox === to.definition.viewBox &&
    from.commands.length > 1 &&
    from.topology === to.topology
  );
}

export function interpolatePath(
  from: NormalizedIconGeometry,
  to: NormalizedIconGeometry,
  progress: number,
) {
  const amount = Math.min(1, Math.max(0, progress));
  return from.commands
    .map(({ type }, index) => {
      const start = from.commands[index]?.point ?? { x: 0, y: 0 };
      const end = to.commands[index]?.point ?? start;
      const x = start.x + (end.x - start.x) * amount;
      const y = start.y + (end.y - start.y) * amount;
      return `${type}${x.toFixed(3)} ${y.toFixed(3)}`;
    })
    .join(" ");
}
