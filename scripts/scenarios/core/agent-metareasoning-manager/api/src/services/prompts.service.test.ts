import 'reflect-metadata';
import { container } from 'tsyringe';
import { PromptsService } from './prompts.service';
import { PromptsRepository } from '../repositories/prompts.repository';

// Mock the repository
jest.mock('../repositories/prompts.repository');

describe('PromptsService', () => {
  let service: PromptsService;
  let mockRepository: jest.Mocked<PromptsRepository>;

  beforeEach(() => {
    container.clearInstances();
    
    // Create mock repository
    mockRepository = {
      findPaginated: jest.fn(),
      findByIdOrThrow: jest.fn(),
      create: jest.fn(),
      update: jest.fn(),
      search: jest.fn(),
      findByCategory: jest.fn(),
      findByTag: jest.fn(),
      incrementVersion: jest.fn()
    } as any;

    // Register mocks in container
    container.registerInstance(PromptsRepository, mockRepository);
    container.registerInstance('RedisClient', Promise.resolve(null));
    
    // Create service
    service = container.resolve(PromptsService);
  });

  describe('getPrompts', () => {
    it('should return paginated prompts', async () => {
      const mockPrompts = [
        { id: '1', name: 'Test Prompt 1' },
        { id: '2', name: 'Test Prompt 2' }
      ];
      
      mockRepository.findPaginated.mockResolvedValue({
        data: mockPrompts,
        total: 2
      });

      const result = await service.getPrompts(undefined, 20, 0);

      expect(result).toEqual({
        prompts: mockPrompts,
        total: 2
      });
      expect(mockRepository.findPaginated).toHaveBeenCalledWith(
        20,
        0,
        { status: 'active' }
      );
    });

    it('should filter by category when provided', async () => {
      mockRepository.findPaginated.mockResolvedValue({
        data: [],
        total: 0
      });

      await service.getPrompts('decision', 10, 0);

      expect(mockRepository.findPaginated).toHaveBeenCalledWith(
        10,
        0,
        { category: 'decision', status: 'active' }
      );
    });
  });

  describe('createPrompt', () => {
    it('should create a new prompt', async () => {
      const promptData = {
        name: 'New Prompt',
        category: 'analysis' as const,
        template: 'This is a test template'
      };

      const createdPrompt = {
        id: 'generated-id',
        ...promptData,
        status: 'active',
        version: 1,
        created_at: new Date(),
        updated_at: new Date()
      };

      mockRepository.create.mockResolvedValue(createdPrompt as any);

      const result = await service.createPrompt(promptData);

      expect(result).toEqual(createdPrompt);
      expect(mockRepository.create).toHaveBeenCalledWith(
        expect.objectContaining({
          name: promptData.name,
          category: promptData.category,
          template: promptData.template,
          status: 'active',
          version: 1
        })
      );
    });
  });

  describe('updatePrompt', () => {
    it('should update an existing prompt', async () => {
      const promptId = 'test-id';
      const updateData = {
        name: 'Updated Name'
      };

      mockRepository.findByIdOrThrow.mockResolvedValue({
        id: promptId,
        name: 'Old Name'
      } as any);

      mockRepository.update.mockResolvedValue({
        id: promptId,
        ...updateData
      } as any);

      const result = await service.updatePrompt(promptId, updateData);

      expect(result.name).toBe(updateData.name);
      expect(mockRepository.update).toHaveBeenCalledWith(promptId, updateData);
    });

    it('should increment version when template is updated', async () => {
      const promptId = 'test-id';
      const updateData = {
        template: 'Updated template'
      };

      mockRepository.findByIdOrThrow.mockResolvedValue({} as any);
      mockRepository.incrementVersion.mockResolvedValue({} as any);
      mockRepository.update.mockResolvedValue({} as any);

      await service.updatePrompt(promptId, updateData);

      expect(mockRepository.incrementVersion).toHaveBeenCalledWith(promptId);
    });
  });

  describe('searchPrompts', () => {
    it('should search prompts by term', async () => {
      const searchTerm = 'decision';
      const mockResults = [
        { id: '1', name: 'Decision Framework' }
      ];

      mockRepository.search.mockResolvedValue(mockResults as any);

      const result = await service.searchPrompts(searchTerm);

      expect(result).toEqual(mockResults);
      expect(mockRepository.search).toHaveBeenCalledWith(searchTerm);
    });
  });
});