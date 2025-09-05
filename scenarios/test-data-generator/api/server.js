const express = require('express');
const cors = require('cors');
const { faker } = require('@faker-js/faker');
const { v4: uuidv4 } = require('uuid');
const Joi = require('joi');
const createCsvWriter = require('csv-writer').createObjectCsvWriter;
const js2xmlparser = require('js2xmlparser');

const app = express();
const PORT = process.env.API_PORT || 3001;

// Middleware
app.use(cors());
app.use(express.json());

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'test-data-generator-api',
    timestamp: new Date().toISOString(),
    uptime: process.uptime()
  });
});

// Data type definitions
const dataTypes = {
  users: {
    name: 'Users',
    description: 'Generate user profiles with personal information',
    fields: ['id', 'name', 'email', 'phone', 'address', 'birthdate', 'avatar']
  },
  companies: {
    name: 'Companies',
    description: 'Generate company information',
    fields: ['id', 'name', 'industry', 'website', 'email', 'phone', 'address']
  },
  products: {
    name: 'Products',
    description: 'Generate product listings',
    fields: ['id', 'name', 'description', 'price', 'category', 'sku', 'inStock']
  },
  orders: {
    name: 'Orders',
    description: 'Generate order data',
    fields: ['id', 'userId', 'total', 'status', 'items', 'createdAt']
  }
};

// Validation schemas
const generateSchema = Joi.object({
  count: Joi.number().integer().min(1).max(10000).default(10),
  format: Joi.string().valid('json', 'csv', 'xml', 'sql').default('json'),
  fields: Joi.array().items(Joi.string()).optional(),
  seed: Joi.string().optional()
});

const customSchema = Joi.object({
  count: Joi.number().integer().min(1).max(10000).default(10),
  format: Joi.string().valid('json', 'csv', 'xml', 'sql').default('json'),
  schema: Joi.object().required(),
  seed: Joi.string().optional()
});

// Data generators
const generators = {
  users: (count, fields = null, seed = null) => {
    if (seed) faker.seed(parseInt(seed));
    
    const defaultFields = ['id', 'name', 'email', 'phone', 'address'];
    const selectedFields = fields || defaultFields;
    
    return Array.from({ length: count }, () => {
      const user = {};
      
      selectedFields.forEach(field => {
        switch (field) {
          case 'id':
            user.id = uuidv4();
            break;
          case 'name':
            user.name = faker.person.fullName();
            break;
          case 'email':
            user.email = faker.internet.email();
            break;
          case 'phone':
            user.phone = faker.phone.number();
            break;
          case 'address':
            user.address = {
              street: faker.location.streetAddress(),
              city: faker.location.city(),
              state: faker.location.state(),
              zipCode: faker.location.zipCode(),
              country: faker.location.country()
            };
            break;
          case 'birthdate':
            user.birthdate = faker.date.birthdate();
            break;
          case 'avatar':
            user.avatar = faker.image.avatar();
            break;
        }
      });
      
      return user;
    });
  },

  companies: (count, fields = null, seed = null) => {
    if (seed) faker.seed(parseInt(seed));
    
    const defaultFields = ['id', 'name', 'industry', 'website', 'email'];
    const selectedFields = fields || defaultFields;
    
    return Array.from({ length: count }, () => {
      const company = {};
      
      selectedFields.forEach(field => {
        switch (field) {
          case 'id':
            company.id = uuidv4();
            break;
          case 'name':
            company.name = faker.company.name();
            break;
          case 'industry':
            company.industry = faker.commerce.department();
            break;
          case 'website':
            company.website = faker.internet.url();
            break;
          case 'email':
            company.email = faker.internet.email();
            break;
          case 'phone':
            company.phone = faker.phone.number();
            break;
          case 'address':
            company.address = {
              street: faker.location.streetAddress(),
              city: faker.location.city(),
              state: faker.location.state(),
              zipCode: faker.location.zipCode(),
              country: faker.location.country()
            };
            break;
        }
      });
      
      return company;
    });
  },

  products: (count, fields = null, seed = null) => {
    if (seed) faker.seed(parseInt(seed));
    
    const defaultFields = ['id', 'name', 'price', 'category'];
    const selectedFields = fields || defaultFields;
    
    return Array.from({ length: count }, () => {
      const product = {};
      
      selectedFields.forEach(field => {
        switch (field) {
          case 'id':
            product.id = uuidv4();
            break;
          case 'name':
            product.name = faker.commerce.productName();
            break;
          case 'description':
            product.description = faker.commerce.productDescription();
            break;
          case 'price':
            product.price = parseFloat(faker.commerce.price());
            break;
          case 'category':
            product.category = faker.commerce.department();
            break;
          case 'sku':
            product.sku = faker.string.alphanumeric(8).toUpperCase();
            break;
          case 'inStock':
            product.inStock = faker.datatype.boolean();
            break;
        }
      });
      
      return product;
    });
  },

  custom: (count, schema, seed = null) => {
    if (seed) faker.seed(parseInt(seed));
    
    return Array.from({ length: count }, () => {
      const item = {};
      
      Object.entries(schema).forEach(([field, type]) => {
        switch (type) {
          case 'string':
            item[field] = faker.lorem.words(3);
            break;
          case 'integer':
            item[field] = faker.number.int({ min: 1, max: 1000 });
            break;
          case 'decimal':
            item[field] = parseFloat(faker.commerce.price());
            break;
          case 'boolean':
            item[field] = faker.datatype.boolean();
            break;
          case 'email':
            item[field] = faker.internet.email();
            break;
          case 'phone':
            item[field] = faker.phone.number();
            break;
          case 'date':
            item[field] = faker.date.recent();
            break;
          case 'uuid':
            item[field] = uuidv4();
            break;
          default:
            item[field] = faker.lorem.word();
        }
      });
      
      return item;
    });
  }
};

// Format data based on requested format
const formatData = (data, format) => {
  switch (format) {
    case 'json':
      return { data, format: 'json' };
    case 'csv':
      // This would need to be handled differently for actual CSV generation
      return { data, format: 'csv', note: 'CSV formatting requires file download' };
    case 'xml':
      return { data: js2xmlparser.parse('items', { item: data }), format: 'xml' };
    case 'sql':
      // Basic SQL INSERT generation
      if (data.length === 0) return { data: '', format: 'sql' };
      
      const tableName = 'generated_data';
      const columns = Object.keys(data[0]);
      const values = data.map(item => 
        `(${columns.map(col => typeof item[col] === 'string' ? `'${item[col]}'` : item[col]).join(', ')})`
      ).join(',\n  ');
      
      const sqlQuery = `INSERT INTO ${tableName} (${columns.join(', ')}) VALUES\n  ${values};`;
      return { data: sqlQuery, format: 'sql' };
    default:
      return { data, format: 'json' };
  }
};

// Routes
app.get('/api/types', (req, res) => {
  res.json({
    types: Object.keys(dataTypes),
    definitions: dataTypes
  });
});

// Generate predefined data types
Object.keys(dataTypes).forEach(type => {
  app.post(`/api/generate/${type}`, (req, res) => {
    try {
      const { error, value } = generateSchema.validate(req.body);
      if (error) {
        return res.status(400).json({ error: error.details[0].message });
      }

      const { count, format, fields, seed } = value;
      const data = generators[type](count, fields, seed);
      const formattedData = formatData(data, format);

      res.json({
        success: true,
        type,
        count: data.length,
        timestamp: new Date().toISOString(),
        ...formattedData
      });
    } catch (err) {
      res.status(500).json({ 
        success: false, 
        error: 'Failed to generate data',
        details: err.message 
      });
    }
  });
});

// Generate custom schema data
app.post('/api/generate/custom', (req, res) => {
  try {
    const { error, value } = customSchema.validate(req.body);
    if (error) {
      return res.status(400).json({ error: error.details[0].message });
    }

    const { count, format, schema, seed } = value;
    const data = generators.custom(count, schema, seed);
    const formattedData = formatData(data, format);

    res.json({
      success: true,
      type: 'custom',
      schema,
      count: data.length,
      timestamp: new Date().toISOString(),
      ...formattedData
    });
  } catch (err) {
    res.status(500).json({ 
      success: false, 
      error: 'Failed to generate custom data',
      details: err.message 
    });
  }
});

// Error handling
app.use((err, req, res, next) => {
  console.error(err.stack);
  res.status(500).json({
    success: false,
    error: 'Internal server error',
    message: process.env.NODE_ENV === 'development' ? err.message : 'Something went wrong'
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    success: false,
    error: 'Endpoint not found',
    path: req.path
  });
});

// Start server
const server = app.listen(PORT, () => {
  console.log(`Test Data Generator API running on port ${PORT}`);
  console.log(`Health check available at: http://localhost:${PORT}/health`);
});

module.exports = { app, server };