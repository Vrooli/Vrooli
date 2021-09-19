import { TABLES } from "../../db";
import { SKU_STATUS } from "@local/shared";
import pkg from '@prisma/client';
const { PrismaClient } = pkg;

const prisma = new PrismaClient()

// Reads an .xls or .xlsx availability file into the database.
// SKUs of products not in the availability file will be hidden
export async function uploadAvailabilityProcess(job) {
    console.info('📊 Updating availability...')
    console.info('SKUs not in the availability data will be hidden');

    const rows = job.data.rows;
    const header = rows[0];
    const content = rows.slice(1, rows.length);
    // Determine which columns data is in
    const index = {
        name: header.indexOf('Name'),
        size: header.indexOf('Size'),
        note: header.indexOf('Notes'),
        price: header.indexOf('Price'),
        sku: header.indexOf('SKU'),
        availability: header.indexOf('Quantity')
    }
    // Hide all existing SKUs, so only the SKUs in this file can be set to visible
    await prisma[TABLES.Sku].updateMany({ data: { status: SKU_STATUS.Inactive } })
    for (const row of content) {
        // Insert or update product data from row
        const name = row[index.name];
        let product = await prisma[TABLES.Product].findUnique({ where: { name }, select: {
            id: true,
            traits: { select: { id: true, name: true, value: true } }
        } });
        if (!product) {
            console.info(`Creating new product: ${name}`);
            product = await prisma[TABLES.Product].create({ data: { name } });
        }
        // If traits don't exist, replace with empty array
        if (!Array.isArray(product.traits)) product.traits = [];
        // Upsert traits
        for (const key of ['name']) {
            if (row[index[key]]) {
                try {
                    const updateData = { productId: product.id, name: key, value: row[index[key]] };
                    await prisma[TABLES.ProductTrait].upsert({
                        where: { product_trait_productid_name_unique: { productId: product.id, name: key }},
                        update: updateData,
                        create: updateData
                    })
                } catch(error) { console.error(error)}
            }
        }
        // Insert or update SKU data from row
        const sku_data = {
            sku: row[index.sku] ?? '',
            size: parseFloat((row[index.size]+'').replace(/\D/g, '')) || null, //'#3.5' -> 3.5
            price: parseFloat((row[index.price]+'').replace(/[^\d.-]/g, '')) || null, //'$23.32' -> 23.32
            note: row[index.note],
            availability: parseInt(row[index.availability]) || 0,
            productId: product.id,
            status: SKU_STATUS.Active
        }
        if (!sku_data.sku) {
            console.error('⛔️ Cannot update rows without a SKU');
            continue;
        }
        try {
            await prisma[TABLES.Sku].upsert({
                where: { sku: sku_data.sku },
                update: sku_data,
                create: sku_data
            })
        } catch(error) { console.error(error) }
    }

    console.info('✅ Availability updated!')
}