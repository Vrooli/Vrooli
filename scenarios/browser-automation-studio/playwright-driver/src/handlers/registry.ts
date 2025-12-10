import type { InstructionHandler } from './base';
import type { CompiledInstruction } from '../types';
import { UnsupportedInstructionError, logger, normalizeInstructionType } from '../utils';

/**
 * Handler registry
 *
 * Manages registration and lookup of instruction handlers
 */
export class HandlerRegistry {
  private handlers: Map<string, InstructionHandler> = new Map();

  /**
   * Register a handler
   *
   * Hardened: Normalizes instruction types to lowercase on registration
   * to ensure consistent lookup regardless of case in source definitions.
   */
  register(handler: InstructionHandler): void {
    const types = handler.getSupportedTypes();

    for (const rawType of types) {
      // Hardened: Normalize type on registration for consistent lookup
      const type = normalizeInstructionType(rawType);

      if (this.handlers.has(type)) {
        logger.warn('Handler already registered for type, overwriting', {
          type,
          rawType,
          handler: handler.constructor.name,
        });
      }

      this.handlers.set(type, handler);
      logger.debug('Registered handler', {
        type,
        handler: handler.constructor.name,
      });
    }
  }

  /**
   * Get handler for instruction type
   */
  getHandler(instruction: CompiledInstruction): InstructionHandler {
    // Normalize instruction type to lowercase for case-insensitive lookup
    const normalizedType = instruction.type.toLowerCase();
    const handler = this.handlers.get(normalizedType);

    if (!handler) {
      throw new UnsupportedInstructionError(instruction.type);
    }

    return handler;
  }

  /**
   * Check if type is supported
   */
  isSupported(type: string): boolean {
    return this.handlers.has(type.toLowerCase());
  }

  /**
   * Get all supported types
   */
  getSupportedTypes(): string[] {
    return Array.from(this.handlers.keys());
  }

  /**
   * Get count of registered handlers
   */
  getHandlerCount(): number {
    return this.handlers.size;
  }
}

/**
 * Global handler registry instance
 */
export const handlerRegistry = new HandlerRegistry();
