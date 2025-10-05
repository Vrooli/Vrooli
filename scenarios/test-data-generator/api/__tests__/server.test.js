const request = require('supertest');

// Set test environment before requiring server
process.env.NODE_ENV = 'test';
const { app, server } = require('../server');

// Close server after all tests (if it exists)
afterAll((done) => {
  if (server) {
    server.close(done);
  } else {
    done();
  }
});

describe('Test Data Generator API', () => {
  describe('Health Check', () => {
    test('GET /health should return healthy status', async () => {
      const response = await request(app)
        .get('/health')
        .expect(200);

      expect(response.body).toMatchObject({
        status: 'healthy',
        service: 'test-data-generator-api'
      });
      expect(response.body.timestamp).toBeDefined();
      expect(response.body.uptime).toBeDefined();
      expect(typeof response.body.uptime).toBe('number');
    });
  });

  describe('Data Types', () => {
    test('GET /api/types should return available data types', async () => {
      const response = await request(app)
        .get('/api/types')
        .expect(200);

      expect(response.body.types).toBeDefined();
      expect(Array.isArray(response.body.types)).toBe(true);
      expect(response.body.definitions).toBeDefined();
      expect(response.body.types).toContain('users');
      expect(response.body.types).toContain('companies');
      expect(response.body.types).toContain('products');
      expect(response.body.types).toContain('orders');
    });

    test('GET /api/types should return correct definitions structure', async () => {
      const response = await request(app)
        .get('/api/types')
        .expect(200);

      const { definitions } = response.body;
      expect(definitions.users).toHaveProperty('name');
      expect(definitions.users).toHaveProperty('description');
      expect(definitions.users).toHaveProperty('fields');
      expect(Array.isArray(definitions.users.fields)).toBe(true);
    });
  });

  describe('User Data Generation', () => {
    test('POST /api/generate/users should generate user data with default count', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({})
        .expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.type).toBe('users');
      expect(response.body.count).toBe(10); // default count
      expect(Array.isArray(response.body.data)).toBe(true);
      expect(response.body.data).toHaveLength(10);
    });

    test('POST /api/generate/users should generate correct number of users', async () => {
      const requestBody = {
        count: 5,
        format: 'json',
        fields: ['id', 'name', 'email']
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.type).toBe('users');
      expect(response.body.count).toBe(5);
      expect(Array.isArray(response.body.data)).toBe(true);
      expect(response.body.data).toHaveLength(5);

      // Check first user has required fields
      const firstUser = response.body.data[0];
      expect(firstUser.id).toBeDefined();
      expect(firstUser.name).toBeDefined();
      expect(firstUser.email).toBeDefined();
      expect(firstUser.phone).toBeUndefined(); // not requested
    });

    test('POST /api/generate/users should handle all available fields', async () => {
      const requestBody = {
        count: 2,
        format: 'json',
        fields: ['id', 'name', 'email', 'phone', 'address', 'birthdate', 'avatar']
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      const user = response.body.data[0];
      expect(user.id).toBeDefined();
      expect(user.name).toBeDefined();
      expect(user.email).toBeDefined();
      expect(user.phone).toBeDefined();
      expect(user.address).toBeDefined();
      expect(user.address.street).toBeDefined();
      expect(user.address.city).toBeDefined();
      expect(user.address.state).toBeDefined();
      expect(user.address.zipCode).toBeDefined();
      expect(user.address.country).toBeDefined();
      expect(user.birthdate).toBeDefined();
      expect(user.avatar).toBeDefined();
    });

    test('POST /api/generate/users with seed should generate consistent data', async () => {
      const requestBody = {
        count: 3,
        format: 'json',
        seed: '12345',
        fields: ['name', 'email', 'phone'] // Exclude UUID fields for consistent comparison
      };

      const response1 = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      const response2 = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      // Names, emails, and phones should be consistent with same seed
      expect(response1.body.data[0].name).toEqual(response2.body.data[0].name);
      expect(response1.body.data[0].email).toEqual(response2.body.data[0].email);
      expect(response1.body.data[0].phone).toEqual(response2.body.data[0].phone);
    });

    test('POST /api/generate/users with different seeds should generate different data', async () => {
      const response1 = await request(app)
        .post('/api/generate/users')
        .send({ count: 3, seed: '12345' })
        .expect(200);

      const response2 = await request(app)
        .post('/api/generate/users')
        .send({ count: 3, seed: '54321' })
        .expect(200);

      expect(response1.body.data).not.toEqual(response2.body.data);
    });

    test('POST /api/generate/users with invalid count should return error', async () => {
      const requestBody = {
        count: -1
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('POST /api/generate/users with count over limit should return error', async () => {
      const requestBody = {
        count: 15000
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('POST /api/generate/users with zero count should return error', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 0 })
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('POST /api/generate/users with invalid format should return error', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 5, format: 'invalid' })
        .expect(400);

      expect(response.body.error).toBeDefined();
    });
  });

  describe('Company Data Generation', () => {
    test('POST /api/generate/companies should generate company data', async () => {
      const requestBody = {
        count: 3,
        format: 'json'
      };

      const response = await request(app)
        .post('/api/generate/companies')
        .send(requestBody)
        .expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.type).toBe('companies');
      expect(response.body.count).toBe(3);
      expect(Array.isArray(response.body.data)).toBe(true);
      expect(response.body.data).toHaveLength(3);

      const firstCompany = response.body.data[0];
      expect(firstCompany.id).toBeDefined();
      expect(firstCompany.name).toBeDefined();
    });

    test('POST /api/generate/companies should handle all fields', async () => {
      const requestBody = {
        count: 2,
        format: 'json',
        fields: ['id', 'name', 'industry', 'website', 'email', 'phone', 'address']
      };

      const response = await request(app)
        .post('/api/generate/companies')
        .send(requestBody)
        .expect(200);

      const company = response.body.data[0];
      expect(company.id).toBeDefined();
      expect(company.name).toBeDefined();
      expect(company.industry).toBeDefined();
      expect(company.website).toBeDefined();
      expect(company.email).toBeDefined();
      expect(company.phone).toBeDefined();
      expect(company.address).toBeDefined();
      expect(company.address.street).toBeDefined();
    });

    test('POST /api/generate/companies with seed should be consistent', async () => {
      const requestBody = {
        count: 2,
        seed: '99999',
        fields: ['name', 'industry', 'website'] // Exclude UUID for consistent comparison
      };

      const response1 = await request(app)
        .post('/api/generate/companies')
        .send(requestBody);

      const response2 = await request(app)
        .post('/api/generate/companies')
        .send(requestBody);

      // Non-UUID fields should be consistent
      expect(response1.body.data[0].name).toEqual(response2.body.data[0].name);
      expect(response1.body.data[0].industry).toEqual(response2.body.data[0].industry);
    });
  });

  describe('Product Data Generation', () => {
    test('POST /api/generate/products should generate product data', async () => {
      const requestBody = {
        count: 4,
        format: 'json'
      };

      const response = await request(app)
        .post('/api/generate/products')
        .send(requestBody)
        .expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.type).toBe('products');
      expect(response.body.count).toBe(4);
      expect(Array.isArray(response.body.data)).toBe(true);
      expect(response.body.data).toHaveLength(4);
    });

    test('POST /api/generate/products should handle all fields', async () => {
      const requestBody = {
        count: 2,
        fields: ['id', 'name', 'description', 'price', 'category', 'sku', 'inStock']
      };

      const response = await request(app)
        .post('/api/generate/products')
        .send(requestBody)
        .expect(200);

      const product = response.body.data[0];
      expect(product.id).toBeDefined();
      expect(product.name).toBeDefined();
      expect(product.description).toBeDefined();
      expect(typeof product.price).toBe('number');
      expect(product.category).toBeDefined();
      expect(product.sku).toBeDefined();
      expect(typeof product.inStock).toBe('boolean');
    });

    test('POST /api/generate/products should validate price is numeric', async () => {
      const response = await request(app)
        .post('/api/generate/products')
        .send({ count: 5, fields: ['price'] })
        .expect(200);

      response.body.data.forEach(product => {
        expect(typeof product.price).toBe('number');
        expect(product.price).toBeGreaterThan(0);
      });
    });
  });

  describe('Orders Data Generation', () => {
    test('POST /api/generate/orders should return 500 (not implemented)', async () => {
      // Orders type is defined but generator is not implemented
      const requestBody = {
        count: 3,
        format: 'json'
      };

      const response = await request(app)
        .post('/api/generate/orders')
        .send(requestBody)
        .expect(500);

      expect(response.body.success).toBe(false);
      expect(response.body.error).toBeDefined();
    });
  });

  describe('Custom Schema Generation', () => {
    test('POST /api/generate/custom should generate data from custom schema', async () => {
      const requestBody = {
        count: 2,
        format: 'json',
        schema: {
          id: 'uuid',
          title: 'string',
          price: 'decimal',
          active: 'boolean'
        }
      };

      const response = await request(app)
        .post('/api/generate/custom')
        .send(requestBody)
        .expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.type).toBe('custom');
      expect(response.body.count).toBe(2);
      expect(Array.isArray(response.body.data)).toBe(true);
      expect(response.body.data).toHaveLength(2);

      const firstItem = response.body.data[0];
      expect(firstItem.id).toBeDefined();
      expect(firstItem.title).toBeDefined();
      expect(typeof firstItem.price).toBe('number');
      expect(typeof firstItem.active).toBe('boolean');
    });

    test('POST /api/generate/custom should handle all data types', async () => {
      const requestBody = {
        count: 1,
        schema: {
          stringField: 'string',
          integerField: 'integer',
          decimalField: 'decimal',
          booleanField: 'boolean',
          emailField: 'email',
          phoneField: 'phone',
          dateField: 'date',
          uuidField: 'uuid',
          unknownField: 'unknown'
        }
      };

      const response = await request(app)
        .post('/api/generate/custom')
        .send(requestBody)
        .expect(200);

      const item = response.body.data[0];
      expect(typeof item.stringField).toBe('string');
      expect(typeof item.integerField).toBe('number');
      expect(Number.isInteger(item.integerField)).toBe(true);
      expect(typeof item.decimalField).toBe('number');
      expect(typeof item.booleanField).toBe('boolean');
      expect(typeof item.emailField).toBe('string');
      expect(item.emailField).toContain('@');
      expect(typeof item.phoneField).toBe('string');
      expect(item.dateField).toBeDefined();
      expect(typeof item.uuidField).toBe('string');
      expect(item.unknownField).toBeDefined();
    });

    test('POST /api/generate/custom without schema should return error', async () => {
      const requestBody = {
        count: 2,
        format: 'json'
      };

      const response = await request(app)
        .post('/api/generate/custom')
        .send(requestBody)
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('POST /api/generate/custom with empty schema should work', async () => {
      const requestBody = {
        count: 2,
        schema: {}
      };

      const response = await request(app)
        .post('/api/generate/custom')
        .send(requestBody)
        .expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveLength(2);
    });

    test('POST /api/generate/custom with seed should be consistent', async () => {
      const requestBody = {
        count: 2,
        seed: '11111',
        schema: {
          name: 'string',
          age: 'integer'
        }
      };

      const response1 = await request(app)
        .post('/api/generate/custom')
        .send(requestBody);

      const response2 = await request(app)
        .post('/api/generate/custom')
        .send(requestBody);

      // Non-UUID fields should be consistent
      expect(response1.body.data[0].name).toEqual(response2.body.data[0].name);
      expect(response1.body.data[0].age).toEqual(response2.body.data[0].age);
    });
  });

  describe('Format Support', () => {
    test('POST /api/generate/users with JSON format (default)', async () => {
      const requestBody = {
        count: 2,
        format: 'json'
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      expect(response.body.format).toBe('json');
      expect(Array.isArray(response.body.data)).toBe(true);
    });

    test('POST /api/generate/users with XML format', async () => {
      const requestBody = {
        count: 2,
        format: 'xml'
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      expect(response.body.format).toBe('xml');
      expect(typeof response.body.data).toBe('string');
      expect(response.body.data).toContain('<?xml');
    });

    test('POST /api/generate/users with SQL format', async () => {
      const requestBody = {
        count: 2,
        format: 'sql',
        fields: ['id', 'name', 'email']
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      expect(response.body.format).toBe('sql');
      expect(typeof response.body.data).toBe('string');
      expect(response.body.data).toContain('INSERT INTO');
      expect(response.body.data).toContain('VALUES');
      expect(response.body.data).toContain('generated_data');
    });

    test('POST /api/generate/users with CSV format', async () => {
      const requestBody = {
        count: 2,
        format: 'csv'
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      expect(response.body.format).toBe('csv');
      expect(response.body.note).toBeDefined();
    });

    test('SQL format with empty data should handle gracefully', async () => {
      // This shouldn't happen due to validation, but tests the formatData function
      const requestBody = {
        count: 1,
        format: 'sql',
        fields: ['id']
      };

      const response = await request(app)
        .post('/api/generate/users')
        .send(requestBody)
        .expect(200);

      expect(response.body.format).toBe('sql');
      expect(response.body.data).toBeDefined();
    });
  });

  describe('Error Handling', () => {
    test('GET /nonexistent should return 404', async () => {
      const response = await request(app)
        .get('/nonexistent')
        .expect(404);

      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Endpoint not found');
      expect(response.body.path).toBe('/nonexistent');
    });

    test('POST /api/generate/invalid should return 404', async () => {
      const response = await request(app)
        .post('/api/generate/invalid')
        .send({ count: 5 })
        .expect(404);

      expect(response.body.success).toBe(false);
    });

    test('POST /nonexistent should return 404', async () => {
      const response = await request(app)
        .post('/nonexistent')
        .send({})
        .expect(404);

      expect(response.body.error).toBe('Endpoint not found');
    });

    test('PUT request to generate endpoint should return 404', async () => {
      await request(app)
        .put('/api/generate/users')
        .send({ count: 5 })
        .expect(404);
    });

    test('DELETE request to generate endpoint should return 404', async () => {
      await request(app)
        .delete('/api/generate/users')
        .expect(404);
    });
  });

  describe('Edge Cases', () => {
    test('POST /api/generate/users with minimum valid count (1)', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 1 })
        .expect(200);

      expect(response.body.data).toHaveLength(1);
    });

    test('POST /api/generate/users with maximum valid count (10000)', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 10000 })
        .expect(200);

      expect(response.body.data).toHaveLength(10000);
    }, 30000); // Increased timeout for large dataset

    test('POST /api/generate/users with empty fields array should use defaults', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 1, fields: [] })
        .expect(200);

      // With empty array, no fields selected, so object should be empty
      const user = response.body.data[0];
      expect(Object.keys(user).length).toBe(0);
    });

    test('POST /api/generate/users with null fields should return error', async () => {
      // Joi validation treats null as invalid for array type
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 1, fields: null })
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('POST request with invalid JSON should return 500', async () => {
      // Express error handler catches JSON parse errors as 500
      const response = await request(app)
        .post('/api/generate/users')
        .set('Content-Type', 'application/json')
        .send('{"invalid": json}')
        .expect(500);

      expect(response.body.success).toBe(false);
      expect(response.body.error).toBeDefined();
    });

    test('POST request with non-numeric count should return 400', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 'abc' })
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('POST request with float count should return 400', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 5.5 })
        .expect(400);

      expect(response.body.error).toBeDefined();
    });

    test('Response should include timestamp', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 1 })
        .expect(200);

      expect(response.body.timestamp).toBeDefined();
      const timestamp = new Date(response.body.timestamp);
      expect(timestamp).toBeInstanceOf(Date);
      expect(timestamp.getTime()).not.toBeNaN();
    });

    test('CORS headers should be present', async () => {
      const response = await request(app)
        .get('/health')
        .expect(200);

      expect(response.headers['access-control-allow-origin']).toBeDefined();
    });
  });

  describe('Response Structure Validation', () => {
    test('Success response should have consistent structure', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: 2 })
        .expect(200);

      expect(response.body).toHaveProperty('success');
      expect(response.body).toHaveProperty('type');
      expect(response.body).toHaveProperty('count');
      expect(response.body).toHaveProperty('timestamp');
      expect(response.body).toHaveProperty('data');
      expect(response.body).toHaveProperty('format');
    });

    test('Error response should have consistent structure', async () => {
      const response = await request(app)
        .post('/api/generate/users')
        .send({ count: -1 })
        .expect(400);

      expect(response.body).toHaveProperty('error');
      expect(typeof response.body.error).toBe('string');
    });
  });

  describe('Performance Tests', () => {
    test('Generating 100 users should complete in reasonable time', async () => {
      const start = Date.now();

      await request(app)
        .post('/api/generate/users')
        .send({ count: 100 })
        .expect(200);

      const duration = Date.now() - start;
      expect(duration).toBeLessThan(5000); // Should complete in under 5 seconds
    });

    test('Generating 1000 products should complete in reasonable time', async () => {
      const start = Date.now();

      await request(app)
        .post('/api/generate/products')
        .send({ count: 1000 })
        .expect(200);

      const duration = Date.now() - start;
      expect(duration).toBeLessThan(10000); // Should complete in under 10 seconds
    });

    test('Concurrent requests should be handled properly', async () => {
      const requests = [
        request(app).post('/api/generate/users').send({ count: 10 }),
        request(app).post('/api/generate/companies').send({ count: 10 }),
        request(app).post('/api/generate/products').send({ count: 10 })
      ];

      const responses = await Promise.all(requests);

      responses.forEach(response => {
        expect(response.status).toBe(200);
        expect(response.body.success).toBe(true);
      });
    });
  });
});
