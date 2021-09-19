import React, { useState, useEffect } from "react";
import { useParams, useHistory } from "react-router-dom";
import PropTypes from "prop-types";
import { productsQuery } from 'graphql/query';
import { upsertOrderItemMutation } from 'graphql/mutation';
import { useQuery, useMutation } from '@apollo/client';
import { getProductTrait, LINKS, SORT_OPTIONS } from "utils";
import {
    ProductCard,
    ProductDialog
} from 'components';
import { makeStyles } from '@material-ui/styles';
import { mutationWrapper } from "graphql/utils/wrappers";

const useStyles = makeStyles(() => ({
    root: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
        alignItems: 'stretch',
    },
}));

function ShoppingList({
    session,
    onSessionUpdate,
    cart,
    sortBy = SORT_OPTIONS[0].value,
    filters,
    searchString = '',
}) {
    const classes = useStyles();
    // Product data for all visible products (i.e. not filtered)
    const [products, setProducts] = useState([]);
    const track_scrolling_id = 'scroll-tracked';
    let history = useHistory();
    const urlParams = useParams();
    // Find current product and current sku
    const currProduct = Array.isArray(products) ? products.find(p => p.skus.some(s => s.sku === urlParams.sku)) : null;
    const currSku = currProduct?.skus ? currProduct.skus.find(s => s.sku === urlParams.sku) : null;
    const { data: productData } = useQuery(productsQuery,  { variables: { sortBy, searchString, active: true } });
    const [upsertOrderItem] = useMutation(upsertOrderItemMutation);

    // useHotkeys('Escape', () => setCurrSku([null, null, null]));

    // Determine which skus will be visible to the customer (i.e. not filtered out)
    useEffect(() => {
        if (!filters || Object.values(filters).length === 0) {
            setProducts(productData?.products);
            return;
        }
        let filtered_products = [];
        for (const product of productData?.products) {
            let found = false;
            for (const [key, value] of Object.entries(filters)) {
                if (found) break;
                const traitValue = getProductTrait(key, product);
                if (traitValue && traitValue.toLowerCase() === (value+'').toLowerCase()) {
                    found = true;
                    break;
                }
                if (!Array.isArray(product.skus)) continue;
                for (let i = 0; i < product.skus.length; i++) {
                    const skuValue = product.skus[i][key];
                    if (skuValue && skuValue.toLowerCase() === (value+'').toLowerCase()) {
                        found = true;
                        break;
                    }
                }
            }
            if (found) filtered_products.push(product);
        }
        setProducts(filtered_products);
    }, [productData, filters, searchString])

    const expandSku = (sku) => {
        history.push(LINKS.Shopping + "/" + sku);
    };

    const toCart = () => {
        history.push(LINKS.Cart);
    }

    const addToCart = (name, sku, quantity) => {
        if (!session?.id) return;
        let max_quantity = parseInt(sku.availability);
        if (Number.isInteger(max_quantity) && quantity > max_quantity) {
            alert(`Error: Cannot add more than ${max_quantity}!`);
            return;
        }
        mutationWrapper({
            mutation: upsertOrderItem,
            data: { variables: { quantity, orderId: cart?.id, skuId: sku.id } },
            successCondition: (response) => response.data.upsertOrderItem,
            onSuccess: () => onSessionUpdate(),
            successMessage: () => `${quantity} ${name}(s) added to cart.`,
            successData: { buttonText: 'View Cart', buttonClicked: toCart },
        })
    }

    return (
        <div className={classes.root} id={track_scrolling_id}>
            {(currProduct) ? <ProductDialog
                onSessionUpdate
                product={currProduct}
                selectedSku={currSku}
                onAddToCart={addToCart}
                open={currProduct !== null}
                onClose={() => history.goBack()} /> : null}
            
            {products?.map((item, index) =>
                <ProductCard key={index}
                    onClick={(data) => expandSku(data.selectedSku?.sku)}
                    product={item} />)}
        </div>
    );
}

ShoppingList.propTypes = {
    session: PropTypes.object,
    onSessionUpdate: PropTypes.func.isRequired,
    cart: PropTypes.object,
    sortBy: PropTypes.string,
    filters: PropTypes.object,
    searchString: PropTypes.string,
};

export { ShoppingList };