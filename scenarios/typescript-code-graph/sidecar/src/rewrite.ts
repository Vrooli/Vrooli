// Filesystem rewrites via ts-morph. Never spawns git/tsc/pnpm — asserted
// by tests/no-external-command.test.ts.

import * as path from "node:path";

import { Project } from "ts-morph";

import type { OperationResult, RewriteOperation } from "./protocol.js";

export class NoTsConfigForRewriteError extends Error {
  readonly kind = "no_tsconfig_found" as const;
}

/**
 * Apply a batch of file moves and import rewrites against the consumer's
 * TS project. Each op's success/failure is reported per-op; partial-failure
 * is by design (per the PRD: filesystem rollback is a non-goal).
 *
 * If `_project` is provided (test seam), it's used directly and `save` is
 * still called on it.
 */
export async function applyRewrite(args: {
  scenarioPath: string;
  operations: RewriteOperation[];
  _project?: Project;
}): Promise<OperationResult[]> {
  const project =
    args._project ??
    new Project({
      tsConfigFilePath: path.join(args.scenarioPath, "tsconfig.json"),
      skipAddingFilesFromTsConfig: false,
    });

  const results: OperationResult[] = [];

  for (const op of args.operations) {
    if (op.file_move) {
      results.push(applyFileMove(project, args.scenarioPath, op.file_move));
    } else if (op.import_rewrite) {
      results.push(
        applyImportRewrite(project, op.import_rewrite),
      );
    } else {
      results.push({
        status: "OPERATION_STATUS_FAILED",
        message: "operation must set either file_move or import_rewrite",
      });
    }
  }

  await project.save();

  return results;
}

function applyFileMove(
  project: Project,
  scenarioPath: string,
  op: { from_path: string; to_path: string },
): OperationResult {
  try {
    const absFrom = path.isAbsolute(op.from_path)
      ? op.from_path
      : path.resolve(scenarioPath, op.from_path);
    const absTo = path.isAbsolute(op.to_path)
      ? op.to_path
      : path.resolve(scenarioPath, op.to_path);
    const sf = project.getSourceFile(absFrom);
    if (!sf) {
      return {
        status: "OPERATION_STATUS_FAILED",
        message: `source file not found in project: ${op.from_path}`,
      };
    }
    sf.move(absTo);
    return { status: "OPERATION_STATUS_OK", message: "" };
  } catch (err) {
    return {
      status: "OPERATION_STATUS_FAILED",
      message: `file_move failed: ${(err as Error).message}`,
    };
  }
}

function applyImportRewrite(
  project: Project,
  op: { old_path: string; new_path: string },
): OperationResult {
  try {
    let touched = 0;
    for (const sf of project.getSourceFiles()) {
      let fileChanged = false;
      for (const imp of sf.getImportDeclarations()) {
        if (imp.getModuleSpecifierValue() === op.old_path) {
          imp.setModuleSpecifier(op.new_path);
          fileChanged = true;
        }
      }
      for (const exp of sf.getExportDeclarations()) {
        if (exp.getModuleSpecifierValue() === op.old_path) {
          exp.setModuleSpecifier(op.new_path);
          fileChanged = true;
        }
      }
      if (fileChanged) touched++;
    }
    return {
      status: "OPERATION_STATUS_OK",
      message: touched > 0 ? `rewrote imports in ${touched} file(s)` : "",
    };
  } catch (err) {
    return {
      status: "OPERATION_STATUS_FAILED",
      message: `import_rewrite failed: ${(err as Error).message}`,
    };
  }
}
