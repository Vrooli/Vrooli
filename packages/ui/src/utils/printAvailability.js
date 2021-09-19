import { jsPDF } from "jspdf";
import 'jspdf-autotable';
import { showPrice, PUBS, PubSub } from 'utils';
import { skusQuery } from 'graphql/query';
import { initializeApollo } from 'graphql/utils/initialize';
import { getProductTrait } from "./productTools";
import { SKU_SORT_OPTIONS } from '@local/shared';

const TITLE_FONT_SIZE = 30;
const LIST_FONT_SIZE = 24;

const centeredText = (text, doc, y) => {
    let textWidth = doc.getStringUnitWidth(text) * doc.internal.getFontSize() / doc.internal.scaleFactor;
    let textOffset = (doc.internal.pageSize.width - textWidth) / 2;
    doc.text(textOffset, y, text);
}

const skusToTable = (skus, priceVisible) => {
    return skus.map(sku => {
        const displayName = sku.product?.name ?? sku.sku;
        const size = isNaN(sku.size) ? sku.size : `#${sku.size}`;
        const availability = sku.availability ?? 'N/A';
        const price = showPrice(sku.price);
        if (priceVisible) return [displayName, size, availability, price]
        return [displayName, size, availability];
    });
}

export const printAvailability = (session, title) => {
    const client = initializeApollo();
    client.query({
        query: skusQuery,
        sortBy: SKU_SORT_OPTIONS.AZ
    }).then(response => {
        const data = response.data.skus;
        const priceVisible = session !== null;
        const table_data = skusToTable(data, priceVisible);
        // Default export is a4 paper, portrait, using millimeters for units
        const doc = new jsPDF();
        doc.setFontSize(TITLE_FONT_SIZE);
        centeredText(title, doc, 10);
        let date = new Date();
        centeredText(`Availability: ${date.toDateString()}`, doc, 20);
        doc.setFontSize(LIST_FONT_SIZE);
        let header = showPrice ? [['Product', 'Size', 'Availability', 'Price']] : [['Product', 'Size', 'Availability']]
        doc.autoTable({
            margin: { top: 30 },
            head: header,
            body: table_data,
        })
        let windowReference = window.open();
        let blob = doc.output('blob', {filename:`availability_${date.getDay()}-${date.getMonth()}-${date.getFullYear()}.pdf`});
        windowReference.location = URL.createObjectURL(blob);
    }).catch(error => {
        PubSub.publish(PUBS.Snack, {message: 'Failed to load inventory.', severity: 'error', data: error });
    });
}
