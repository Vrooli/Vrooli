// Abstract class for providing base functionality to models
// (i.e. create, update, delete)

export abstract class BaseModel<T> {
    // Create a new model
    abstract create(data: T): Promise<T>;
    
    // Update an existing model
    abstract update(id: string, data: T): Promise<T>;
    
    // Delete a model
    abstract delete(id: string): Promise<void>;

    // Delete multiple models
    abstract deleteMany(ids: string[]): Promise<number>;
}